package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/tidwall/gjson"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// ExecContext holds the context for a single step execution
	ExecContext struct {
		engine *Engine
		step   *api.Step
		inputs api.Args
		flowID api.FlowID
		stepID api.StepID
	}

	// MultiArgs maps attribute names to value arrays for parallel execution
	MultiArgs map[api.Name][]any
)

var (
	ErrStepAlreadyPending  = errors.New("step not pending")
	ErrUnsupportedStepType = errors.New("unsupported step type")
)

func (e *Engine) evaluateStepPredicate(
	ctx context.Context, fs FlowStep, step *api.Step, inputs api.Args,
) bool {
	if step.Predicate == nil {
		return true
	}

	comp, err := e.GetCompiledPredicate(fs)
	if err != nil {
		e.failPredicateEvaluation(ctx, fs,
			"Failed to get compiled predicate",
			"predicate compilation failed", err)
		return false
	}

	if comp == nil {
		return true
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		e.failPredicateEvaluation(ctx, fs,
			"Failed to get script environment for predicate",
			"failed to get script environment", err)
		return false
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		e.failPredicateEvaluation(ctx, fs,
			"Failed to evaluate predicate",
			"predicate evaluation failed", err)
		return false
	}

	return shouldExecute
}

func (e *Engine) failPredicateEvaluation(
	ctx context.Context, fs FlowStep, logMsg, failMsg string, err error,
) {
	slog.Error(logMsg,
		slog.Any("step_id", fs.StepID),
		slog.Any("error", err))

	if failErr := e.FailStepExecution(
		ctx, fs, fmt.Sprintf("%s: %s", failMsg, err.Error()),
	); failErr != nil {
		slog.Error("Failed to record predicate failure",
			slog.Any("error", failErr))
	}
}

func (e *Engine) collectStepInputs(step *api.Step, attrs api.Args) api.Args {
	inputs := api.Args{}

	for name, spec := range step.Attributes {
		if !spec.IsInput() {
			continue
		}

		if value, ok := attrs[name]; ok {
			inputs[name] = value
		} else if !spec.IsRequired() && spec.Default != "" {
			inputs[name] = gjson.Parse(spec.Default).Value()
		}
	}

	return inputs
}

// Work item execution functions

func (e *ExecContext) executeWorkItems(
	ctx context.Context, items map[api.Token]*api.WorkState,
) {
	parallelism := 0
	if e.step.WorkConfig != nil {
		parallelism = e.step.WorkConfig.Parallelism
	}

	sem := make(chan struct{}, max(1, parallelism))

	for token, workItem := range items {
		if workItem.Status != api.WorkPending {
			continue
		}

		fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
		if !e.engine.evaluateStepPredicate(ctx, fs, e.step, workItem.Inputs) {
			continue
		}

		err := e.engine.StartWork(ctx, fs, token, workItem.Inputs)
		if err != nil {
			continue
		}

		go func(token api.Token, workItem *api.WorkState) {
			sem <- struct{}{}
			defer func() { <-sem }()

			e.performWorkItem(ctx, token, workItem)
		}(token, workItem)
	}
}

func (e *ExecContext) executeWorkItem(
	ctx context.Context, token api.Token, workItem *api.WorkState,
) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
	if !e.engine.evaluateStepPredicate(
		ctx, fs, e.step, workItem.Inputs,
	) {
		return
	}

	if err := e.engine.StartWork(ctx, fs, token, workItem.Inputs); err != nil {
		return
	}

	e.performWorkItem(ctx, token, workItem)
}

func (e *ExecContext) performWorkItem(
	ctx context.Context, token api.Token, workItem *api.WorkState,
) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
	outputs, err := e.performWork(ctx, workItem.Inputs, token)

	if err != nil {
		e.handleWorkItemFailure(ctx, token, err)
		return
	}

	if !isAsyncStep(e.step.Type) {
		_ = e.engine.CompleteWork(ctx, fs, token, outputs)
	}
}

func (e *ExecContext) handleWorkItemFailure(
	ctx context.Context, token api.Token, err error,
) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		_ = e.engine.NotCompleteWork(ctx, fs, token, err.Error())
		return
	}

	_ = e.engine.FailWork(ctx, fs, token, err.Error())
}

func (e *ExecContext) performWork(
	ctx context.Context, inputs api.Args, token api.Token,
) (api.Args, error) {
	switch e.step.Type {
	case api.StepTypeScript:
		return e.performScriptWork(inputs)
	case api.StepTypeSync, api.StepTypeAsync:
		return e.performHTTPWork(ctx, inputs, token)
	default:
		return nil, ErrUnsupportedStepType
	}
}

func (e *ExecContext) performScriptWork(inputs api.Args) (api.Args, error) {
	c, err := e.engine.GetCompiledScript(FlowStep{
		FlowID: e.flowID,
		StepID: e.stepID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScriptCompileFailed, err)
	}

	if c == nil {
		return nil, ErrScriptCompileFailed
	}

	return e.executeScript(c, inputs)
}

func (e *ExecContext) performHTTPWork(
	ctx context.Context, inputs api.Args, token api.Token,
) (api.Args, error) {
	metadata := e.buildHTTPMetadataWithToken(token)
	return e.engine.stepClient.Invoke(ctx, e.step, inputs, metadata)
}

func (e *ExecContext) buildHTTPMetadataWithToken(token api.Token) api.Metadata {
	metadata := api.Metadata{
		"flow_id":       e.flowID,
		"step_id":       e.stepID,
		"receipt_token": token,
	}

	if isAsyncStep(e.step.Type) {
		metadata["webhook_url"] = fmt.Sprintf(
			"%s/webhook/%s/%s/%s",
			e.engine.config.WebhookBaseURL, e.flowID, e.stepID, token,
		)
	}

	return metadata
}

func (e *ExecContext) executeScript(
	c Compiled, inputs api.Args,
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

// Helper functions for work item execution

func getMultiArgs(argNames []api.Name, inputs api.Args) MultiArgs {
	multiArgs := MultiArgs{}

	for _, name := range argNames {
		if arr := asArray(inputs[name]); arr != nil {
			multiArgs[name] = arr
		}
	}

	return multiArgs
}

func asArray(value any) []any {
	if value == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	result := gjson.ParseBytes(jsonBytes)
	if !result.IsArray() {
		return nil
	}

	arr := make([]any, 0, len(result.Array()))
	for _, item := range result.Array() {
		arr = append(arr, item.Value())
	}
	return arr
}
