package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/api"
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
	ErrScriptNotCompiled      = errors.New("script step has no compiled script")
	ErrPredicateCompileFailed = errors.New("predicate compilation failed")
	ErrPredicateEnvFailed     = errors.New("failed to get script environment")
	ErrPredicateEvalFailed    = errors.New("predicate evaluation failed")
)

func (e *Engine) evaluateStepPredicate(
	step *api.Step, inputs api.Args,
) (bool, error) {
	if step.Predicate == nil {
		return true, nil
	}

	comp, err := e.scripts.Compile(step, step.Predicate)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateEnvFailed, err)
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateEvalFailed, err)
	}

	return shouldExecute, nil
}

func (tx *flowTx) collectStepInputs(
	step *api.Step, flow *api.FlowState,
) api.Args {
	inputs := api.Args{}
	now := tx.Now()
	ev := tx.newStepEval(step.ID, flow, now)

	for name, attr := range step.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}

		if attr.IsConst() {
			inputs[name] = gjson.Parse(attr.Default).Value()
			continue
		}

		if attr.IsOptional() {
			ready, fallback := ev.optionalFallback(name, attr)
			if !ready {
				continue
			}
			if fallback {
				if attr.Default != "" {
					value := gjson.Parse(attr.Default).Value()
					val := tx.mapper.MapInput(step, name, attr, value)
					paramName := tx.mapper.InputParamName(name, attr)
					inputs[paramName] = val
				}
				continue
			}
		}

		attrVal, ok := flow.Attributes[name]
		if !ok {
			if !attr.IsRequired() && attr.Default != "" {
				value := gjson.Parse(attr.Default).Value()
				val := tx.mapper.MapInput(step, name, attr, value)
				paramName := tx.mapper.InputParamName(name, attr)
				inputs[paramName] = val
				continue
			}
			continue
		}

		val := tx.mapper.MapInput(step, name, attr, attrVal.Value)
		paramName := tx.mapper.InputParamName(name, attr)
		inputs[paramName] = val
	}

	return inputs
}

// Work item execution functions

func (e *ExecContext) executeWorkItems(items api.WorkItems) {
	for token, work := range items {
		if work.Status != api.WorkActive {
			continue
		}

		go e.performWorkItem(token, work)
	}
}

func (e *ExecContext) performWorkItem(
	token api.Token, work *api.WorkState,
) {
	if err := e.performWork(work.Inputs, token); err != nil {
		e.handleWorkItemFailure(token, err)
	}
}

func (e *ExecContext) handleWorkItemFailure(token api.Token, err error) {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		recErr := e.engine.NotCompleteWork(fs, token, err.Error())
		if recErr != nil {
			slog.Error("Failed to record work not completed",
				log.FlowID(e.flowID),
				log.StepID(e.stepID),
				log.Error(recErr))
		}
		return
	}

	recErr := e.engine.FailWork(fs, token, err.Error())
	if recErr != nil {
		slog.Error("Failed to record work failure",
			log.FlowID(e.flowID),
			log.StepID(e.stepID),
			log.Error(recErr))
	}
}

func (e *ExecContext) performWork(inputs api.Args, token api.Token) error {
	switch e.step.Type {
	case api.StepTypeScript:
		return e.performScript(inputs, token)
	case api.StepTypeSync:
		return e.performSyncHTTP(inputs, token)
	case api.StepTypeAsync:
		return e.performAsyncHTTP(inputs, token)
	case api.StepTypeFlow:
		return e.performFlow(inputs, token)
	default:
		panic(fmt.Errorf("%w: %s", ErrUnsupportedStepType, e.step.Type))
	}
}

func (e *ExecContext) performScript(inputs api.Args, token api.Token) error {
	c, err := e.engine.scripts.Compile(e.step, e.step.Script)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScriptCompileFailed, err)
	}

	if c == nil {
		panic(fmt.Errorf("%w: %s", ErrScriptNotCompiled, e.stepID))
	}

	outputs, err := e.executeScript(c, inputs)
	if err != nil {
		return err
	}

	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.CompleteWork(fs, token, outputs)
}

func (e *ExecContext) performSyncHTTP(inputs api.Args, token api.Token) error {
	metadata := e.httpMetaForToken(token)
	outputs, err := e.engine.stepClient.Invoke(e.step, inputs, metadata)
	if err != nil {
		return err
	}

	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.CompleteWork(fs, token, outputs)
}

func (e *ExecContext) performAsyncHTTP(inputs api.Args, token api.Token) error {
	metadata := e.httpMetaForToken(token)
	metadata[api.MetaWebhookURL] = fmt.Sprintf("%s/webhook/%s/%s/%s",
		e.engine.config.WebhookBaseURL, e.flowID, e.stepID, token,
	)
	_, err := e.engine.stepClient.Invoke(e.step, inputs, metadata)
	return err
}

func (e *ExecContext) performFlow(initState api.Args, token api.Token) error {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	_, err := e.engine.StartChildFlow(fs, token, e.step, initState)
	if err != nil {
		return err
	}
	return nil
}

func (e *ExecContext) httpMetaForToken(token api.Token) api.Metadata {
	metadata := maps.Clone(e.meta)
	if metadata == nil {
		metadata = api.Metadata{}
	}

	metadata[api.MetaFlowID] = e.flowID
	metadata[api.MetaStepID] = e.stepID
	metadata[api.MetaReceiptToken] = token
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

func extractScriptResult(result api.Args) any {
	if len(result) == 0 {
		return nil
	}
	if val, ok := result["value"]; ok {
		return val
	}
	if len(result) == 1 {
		for _, v := range result {
			return v
		}
	}
	return result
}
