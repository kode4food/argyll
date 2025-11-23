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

// EnqueueStepResult completes a step execution with the provided outputs and
// sets all output attributes in the flow state
func (e *Engine) EnqueueStepResult(
	fs FlowStep, outputs api.Args, dur int64,
) {
	ctx := context.Background()

	if err := e.CompleteStepExecution(ctx, fs, outputs, dur); err != nil {
		slog.Error("Failed to complete step",
			slog.Any("error", err))
		return
	}

	for key, value := range outputs {
		_ = e.SetAttribute(ctx, fs, key, value)
	}
}

func (e *Engine) executeStep(ctx context.Context, fs FlowStep) {
	execCtx := e.PrepareStepExecution(ctx, fs)
	if execCtx == nil {
		return
	}

	execCtx.execute(ctx)
}

// PrepareStepExecution validates and prepares a step for execution, returning
// an execution context or nil if preparation fails
func (e *Engine) PrepareStepExecution(
	ctx context.Context, fs FlowStep,
) *ExecContext {
	if err := e.validateStepExecution(ctx, fs); err != nil {
		slog.Error("Step validation failed",
			slog.Any("flow_id", fs.FlowID),
			slog.Any("step_id", fs.StepID),
			slog.Any("error", err))
		return nil
	}

	_, step, inputs := e.getStepExecutionData(ctx, fs)
	if step == nil {
		return nil
	}

	if !e.shouldExecuteStep(ctx, fs, step, inputs) {
		return nil
	}

	if err := e.StartStepExecution(ctx, fs, step, inputs); err != nil {
		slog.Error("Failed to start step",
			slog.Any("error", err))
		return nil
	}

	return &ExecContext{
		engine: e,
		flowID: fs.FlowID,
		stepID: fs.StepID,
		step:   step,
		inputs: inputs,
	}
}

func (e *Engine) getStepExecutionData(
	ctx context.Context, fs FlowStep,
) (*api.FlowState, *api.Step, api.Args) {
	flow, err := e.GetFlowState(ctx, fs.FlowID)
	if err != nil {
		slog.Error("Failed to get flow state",
			slog.Any("error", err))
		return nil, nil, nil
	}

	step := flow.Plan.GetStep(fs.StepID)
	if step == nil {
		slog.Error("Step not found",
			slog.Any("step_id", fs.StepID))
		return nil, nil, nil
	}

	inputs := e.collectStepInputs(step, flow.GetAttributeArgs())
	return flow, step, inputs
}

func (e *Engine) shouldExecuteStep(
	ctx context.Context, fs FlowStep, step *api.Step, inputs api.Args,
) bool {
	if e.evaluateStepPredicate(ctx, fs, step, inputs) {
		return true
	}

	reason := "predicate evaluated to false"
	slog.Info("Step skipped",
		slog.Any("step_id", fs.StepID),
		slog.Any("flow_id", fs.FlowID),
		slog.String("reason", reason))

	if err := e.SkipStepExecution(ctx, fs, reason); err != nil {
		slog.Error("Failed to skip step",
			slog.Any("error", err))
	}

	return false
}

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
	ctx context.Context, fs FlowStep,
	logMsg, failMsg string, err error,
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

func (e *Engine) validateStepExecution(ctx context.Context, fs FlowStep) error {
	flow, err := e.GetFlowState(ctx, fs.FlowID)
	if err != nil {
		return err
	}

	step := flow.Plan.GetStep(fs.StepID)
	if step == nil {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, fs.StepID)
	}

	exec, ok := flow.Executions[fs.StepID]
	if ok && exec.Status != api.StepPending {
		return fmt.Errorf("%s: %s (status=%s)",
			ErrStepAlreadyPending, fs.StepID, exec.Status)
	}
	return nil
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

func (e *ExecContext) execute(ctx context.Context) {
	flow, err := e.engine.GetFlowState(ctx, e.flowID)
	if err != nil {
		return
	}

	exec, ok := flow.Executions[e.stepID]
	if !ok || exec.WorkItems == nil {
		return
	}

	e.executeWorkItems(ctx, exec.WorkItems)
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

		go func(token api.Token, workItem *api.WorkState) {
			sem <- struct{}{}
			defer func() { <-sem }()

			e.executeWorkItem(ctx, token, workItem)
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
		return e.performScriptWork(ctx, inputs)
	case api.StepTypeSync, api.StepTypeAsync:
		return e.performHTTPWork(ctx, inputs, token)
	default:
		return nil, ErrUnsupportedStepType
	}
}

func (e *ExecContext) performScriptWork(
	_ context.Context, inputs api.Args,
) (api.Args, error) {
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
