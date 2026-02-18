package engine

import (
	"errors"
	"fmt"
	"maps"

	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/api"
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
	ErrStepAlreadyPending  = errors.New("step not pending")
	ErrUnsupportedStepType = errors.New("unsupported step type")
)

func (e *Engine) evaluateStepPredicate(
	step *api.Step, inputs api.Args,
) (bool, error) {
	if step.Predicate == nil {
		return true, nil
	}

	comp, err := e.scripts.Compile(step, step.Predicate)
	if err != nil {
		return false, fmt.Errorf("predicate compilation failed: %w", err)
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		return false, fmt.Errorf("failed to get script environment: %w", err)
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		return false, fmt.Errorf("predicate evaluation failed: %w", err)
	}

	return shouldExecute, nil
}

func (e *Engine) collectStepInputs(step *api.Step, attrs api.Args) api.Args {
	inputs := api.Args{}

	for name, attr := range step.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}

		if attr.IsConst() {
			inputs[name] = gjson.Parse(attr.Default).Value()
			continue
		}

		value, ok := attrs[name]
		if !ok {
			if !attr.IsRequired() && attr.Default != "" {
				value = gjson.Parse(attr.Default).Value()
			} else {
				continue
			}
		}

		val := e.mapper.MapInput(step, name, attr, value)
		paramName := e.mapper.InputParamName(name, attr)
		inputs[paramName] = val
	}

	return inputs
}

// Work item execution functions

func (e *ExecContext) executeWorkItems(items api.WorkItems) {
	for token, workItem := range items {
		if workItem.Status != api.WorkActive {
			continue
		}

		go e.performWorkItem(token, workItem)
	}
}

func (e *ExecContext) performWorkItem(
	token api.Token, workItem *api.WorkState,
) {
	if err := e.performWork(workItem.Inputs, token); err != nil {
		e.handleWorkItemFailure(token, err)
	}
}

func (e *ExecContext) handleWorkItemFailure(token api.Token, err error) {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		_ = e.engine.NotCompleteWork(fs, token, err.Error())
		return
	}

	_ = e.engine.FailWork(fs, token, err.Error())
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
		return ErrUnsupportedStepType
	}
}

func (e *ExecContext) performScript(inputs api.Args, token api.Token) error {
	c, err := e.engine.GetCompiledScript(api.FlowStep{
		FlowID: e.flowID,
		StepID: e.stepID,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScriptCompileFailed, err)
	}

	if c == nil {
		return ErrScriptCompileFailed
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
	mappedState := mapFlowInputs(e.step, initState)
	_, err := e.engine.StartChildFlow(fs, token, e.step, mappedState)
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
