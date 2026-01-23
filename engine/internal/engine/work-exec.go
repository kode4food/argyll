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

		if value, ok := attrs[name]; ok {
			inputs[name] = value
		} else if !attr.IsRequired() && attr.Default != "" {
			inputs[name] = gjson.Parse(attr.Default).Value()
		}
	}

	return inputs
}

// Work item execution functions

func (e *ExecContext) executeWorkItems(items api.WorkItems) {
	parallelism := 0
	if e.step.WorkConfig != nil {
		parallelism = e.step.WorkConfig.Parallelism
	}

	sem := make(chan struct{}, max(1, parallelism))

	for token, workItem := range items {
		if workItem.Status != api.WorkActive {
			continue
		}

		go func(token api.Token, workItem *api.WorkState) {
			sem <- struct{}{}
			defer func() { <-sem }()

			e.performWorkItem(token, workItem)
		}(token, workItem)
	}
}

func (e *ExecContext) executeWorkItem(
	token api.Token, workItem *api.WorkState,
) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
	if workItem.Status != api.WorkActive {
		shouldExecute, err := e.engine.evaluateStepPredicate(
			e.step, workItem.Inputs,
		)
		if err != nil {
			_ = e.engine.FailStepExecution(fs, err.Error())
			return
		}
		if !shouldExecute {
			return
		}
		if err := e.engine.StartWork(fs, token, workItem.Inputs); err != nil {
			return
		}
	}

	e.performWorkItem(token, workItem)
}

func (e *ExecContext) performWorkItem(
	token api.Token, workItem *api.WorkState,
) {
	outputs, err := e.performWork(workItem.Inputs, token)

	if err != nil {
		e.handleWorkItemFailure(token, err)
		return
	}

	if !isAsyncStep(e.step.Type) {
		fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
		_ = e.engine.CompleteWork(fs, token, outputs)
	}
}

func (e *ExecContext) handleWorkItemFailure(token api.Token, err error) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		_ = e.engine.NotCompleteWork(fs, token, err.Error())
		return
	}

	_ = e.engine.FailWork(fs, token, err.Error())
}

func (e *ExecContext) performWork(
	inputs api.Args, token api.Token,
) (api.Args, error) {
	switch e.step.Type {
	case api.StepTypeScript:
		return e.performScriptWork(inputs)
	case api.StepTypeSync, api.StepTypeAsync:
		return e.performHTTPWork(inputs, token)
	case api.StepTypeFlow:
		return e.performFlowWork(inputs, token)
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
	inputs api.Args, token api.Token,
) (api.Args, error) {
	metadata := e.httpMetaForToken(token)
	return e.engine.stepClient.Invoke(e.step, inputs, metadata)
}

func (e *ExecContext) performFlowWork(
	initState api.Args, token api.Token,
) (api.Args, error) {
	fs := FlowStep{FlowID: e.flowID, StepID: e.stepID}
	mappedState := mapFlowInputs(e.step, initState)
	_, err := e.engine.StartChildFlow(fs, token, e.step, mappedState)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (e *ExecContext) httpMetaForToken(token api.Token) api.Metadata {
	metadata := maps.Clone(e.meta)
	if metadata == nil {
		metadata = api.Metadata{}
	}

	metadata[api.MetaFlowID] = e.flowID
	metadata[api.MetaStepID] = e.stepID
	metadata[api.MetaReceiptToken] = token

	if isAsyncStep(e.step.Type) {
		metadata[api.MetaWebhookURL] = fmt.Sprintf(
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
