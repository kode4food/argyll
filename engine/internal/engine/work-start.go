package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// ExecContext holds the context for a single step execution
	ExecContext struct {
		engine *Engine
		step   *api.Step
		inputs api.Args
		flowID api.FlowID
		stepID api.StepID
		meta   api.Metadata
	}

	// MultiArgs maps attribute names to value arrays for parallel execution
	MultiArgs map[api.Name][]any
)

var (
	ErrStepAlreadyPending     = errors.New("step not pending")
	ErrUnsupportedStepType    = errors.New("unsupported step type")
	ErrPredicateCompileFailed = errors.New("predicate compilation failed")
	ErrPredicateEnvFailed     = errors.New("failed to get script environment")
	ErrPredicateEvalFailed    = errors.New("predicate evaluation failed")
)

func (tx *flowTx) handleWorkItemsExecution(
	step *api.Step, inputs api.Args, meta api.Metadata, items api.WorkItems,
) {
	execCtx := &ExecContext{
		engine: tx.Engine,
		flowID: tx.flowID,
		stepID: step.ID,
		step:   step,
		inputs: inputs,
		meta:   meta,
	}
	execCtx.executeWorkItems(items)
}

func (e *ExecContext) executeWorkItems(items api.WorkItems) {
	for token, work := range items {
		if work.Status != api.WorkActive {
			continue
		}

		go e.performWorkItem(token, work)
	}
}

func (e *ExecContext) performWorkItem(tkn api.Token, work *api.WorkState) {
	inputs := e.inputs.Apply(work.Inputs)
	if err := e.performWork(inputs, tkn); err != nil {
		e.handleWorkItemFailure(tkn, err)
	}
}

func (e *ExecContext) handleWorkItemFailure(tkn api.Token, err error) {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		recErr := e.engine.NotCompleteWork(fs, tkn, err.Error())
		if recErr != nil {
			slog.Error("Failed to record work not completed",
				log.FlowID(e.flowID),
				log.StepID(e.stepID),
				log.Error(recErr))
		}
		return
	}

	recErr := e.engine.FailWork(fs, tkn, err.Error())
	if recErr != nil {
		slog.Error("Failed to record work failure",
			log.FlowID(e.flowID),
			log.StepID(e.stepID),
			log.Error(recErr))
	}
}

func (e *ExecContext) performWork(inputs api.Args, tkn api.Token) error {
	switch e.step.Type {
	case api.StepTypeScript:
		return e.performScript(inputs, tkn)
	case api.StepTypeSync:
		return e.performSyncHTTP(inputs, tkn)
	case api.StepTypeAsync:
		return e.performAsyncHTTP(inputs, tkn)
	case api.StepTypeFlow:
		return e.performFlow(inputs, tkn)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedStepType, e.step.Type)
	}
}

func (e *ExecContext) performScript(inputs api.Args, tkn api.Token) error {
	c, err := e.engine.scripts.Compile(e.step, e.step.Script)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScriptCompileFailed, err)
	}

	outputs, err := e.executeScript(c, inputs)
	if err != nil {
		return err
	}

	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.CompleteWork(fs, tkn, outputs)
}

func (e *ExecContext) performSyncHTTP(inputs api.Args, tkn api.Token) error {
	metadata := e.httpMetaForToken(tkn)
	outputs, err := e.engine.stepClient.Invoke(e.step, inputs, metadata)
	if err != nil {
		return err
	}

	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.CompleteWork(fs, tkn, outputs)
}

func (e *ExecContext) performAsyncHTTP(inputs api.Args, tkn api.Token) error {
	metadata := e.httpMetaForToken(tkn)
	metadata[api.MetaWebhookURL] = fmt.Sprintf("%s/webhook/%s/%s/%s",
		e.engine.config.WebhookBaseURL, e.flowID, e.stepID, tkn,
	)
	_, err := e.engine.stepClient.Invoke(e.step, inputs, metadata)
	return err
}

func (e *ExecContext) performFlow(initState api.Args, tkn api.Token) error {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	_, err := e.engine.StartChildFlow(fs, tkn, e.step, initState)
	if err != nil {
		return err
	}
	return nil
}

func (e *ExecContext) httpMetaForToken(tkn api.Token) api.Metadata {
	return e.meta.Apply(api.Metadata{
		api.MetaFlowID:       e.flowID,
		api.MetaStepID:       e.stepID,
		api.MetaReceiptToken: tkn,
	})
}

func (e *ExecContext) executeScript(
	c script.Compiled, inputs api.Args,
) (api.Args, error) {
	language := api.ScriptLangAle
	if e.step.Script != nil {
		language = e.step.Script.Language
	}
	env, err := e.engine.scripts.Get(language)
	if err != nil {
		return nil, err
	}
	return env.ExecuteScript(c, e.step, inputs)
}

func extractScriptResult(result api.Args) any {
	if len(result) == 0 {
		return nil
	}
	if val, ok := result["value"]; ok {
		return val
	}
	if len(result) == 1 {
		for _, value := range result {
			return value
		}
	}
	return result
}

func (tx *flowTx) startPendingWork(step *api.Step) (api.WorkItems, error) {
	stepID := step.ID
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, stepID, exec.Status)
	}

	limit := stepParallelism(step)
	active := countActiveWorkItems(exec.WorkItems)
	remaining := limit - active
	if remaining <= 0 {
		return nil, nil
	}

	now := tx.Now()
	started := api.WorkItems{}
	for token, work := range exec.WorkItems {
		if remaining == 0 {
			break
		}
		shouldStart, err := tx.shouldStartPendingWorkItem(
			step, exec.Inputs, work, now,
		)
		if err != nil {
			return nil, err
		}
		if !shouldStart {
			continue
		}

		inputs := exec.Inputs.Apply(work.Inputs)
		if step.Memoizable {
			if cached, ok := tx.memoCache.Get(step, inputs); ok {
				err := tx.handleMemoCacheHit(stepID, token, cached)
				if err != nil {
					return nil, err
				}
				remaining--
				continue
			}
		}

		if err := tx.raiseWorkStarted(stepID, token, inputs); err != nil {
			return nil, err
		}
		exec = tx.Value().Executions[stepID]
		started[token] = exec.WorkItems[token]
		remaining--
	}

	return started, nil
}

func (tx *flowTx) startRetryWorkItem(
	step *api.Step, tkn api.Token,
) (api.WorkItems, error) {
	stepID := step.ID
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, nil
	}

	work, ok := exec.WorkItems[tkn]
	if !ok {
		return nil, nil
	}

	now := tx.Now()
	shouldStart := false
	switch work.Status {
	case api.WorkPending:
		var err error
		if shouldStart, err = tx.shouldStartRetryPending(
			step, exec.Inputs, work, exec.WorkItems, now,
		); err != nil {
			return nil, err
		}
	case api.WorkFailed:
		if work.NextRetryAt.IsZero() || work.NextRetryAt.After(now) {
			return nil, nil
		}
		shouldStart = true
	case api.WorkActive, api.WorkNotCompleted:
		shouldStart = true
	default:
		return nil, nil
	}
	if !shouldStart {
		return nil, nil
	}

	inputs := exec.Inputs.Apply(work.Inputs)
	if err := tx.raiseWorkStarted(stepID, tkn, inputs); err != nil {
		return nil, err
	}
	exec = tx.Value().Executions[stepID]
	started := api.WorkItems{}
	started[tkn] = exec.WorkItems[tkn]
	return started, nil
}

func (tx *flowTx) shouldStartPendingWorkItem(
	step *api.Step, base api.Args, work *api.WorkState, now time.Time,
) (bool, error) {
	stepID := step.ID
	if work.Status != api.WorkPending {
		return false, nil
	}
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(now) {
		return false, nil
	}
	inputs := base.Apply(work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) shouldStartRetryPending(
	step *api.Step, base api.Args, work *api.WorkState, items api.WorkItems,
	now time.Time,
) (bool, error) {
	stepID := step.ID
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(now) {
		return false, nil
	}
	limit := stepParallelism(step)
	active := countActiveWorkItems(items)
	if active >= limit {
		return false, nil
	}
	inputs := base.Apply(work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) raiseWorkStarted(
	stepID api.StepID, tkn api.Token, inputs api.Args,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
			Inputs: inputs,
		},
	)
}

func stepParallelism(step *api.Step) int {
	if step.WorkConfig == nil || step.WorkConfig.Parallelism <= 0 {
		return 1
	}
	return step.WorkConfig.Parallelism
}

func countActiveWorkItems(items api.WorkItems) int {
	active := 0
	for _, work := range items {
		if work.Status == api.WorkActive {
			active++
		}
	}
	return active
}
