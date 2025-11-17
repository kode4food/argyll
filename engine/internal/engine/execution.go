package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"
	"github.com/tidwall/gjson"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// ExecContext holds the context for a single step execution
	ExecContext struct {
		start  time.Time
		engine *Engine
		step   *api.Step
		inputs api.Args
		flowID timebox.ID
		stepID timebox.ID
	}

	// MultiArgs maps attribute names to value arrays for parallel execution
	MultiArgs map[api.Name][]any
)

var (
	ErrScriptCompileNil    = errors.New("script compilation returned nil")
	ErrStepAlreadyPending  = errors.New("step not pending")
	ErrUnsupportedStepType = errors.New("unsupported step type")
)

// EnqueueStepResult completes a step execution with the provided outputs and
// sets all output attributes in the workflow state
func (e *Engine) EnqueueStepResult(
	flowID, stepID timebox.ID, outputs api.Args, dur int64,
) {
	ctx := context.Background()

	if err := e.CompleteStepExecution(
		ctx, flowID, stepID, outputs, dur,
	); err != nil {
		slog.Error("Failed to complete step",
			slog.Any("error", err))
		return
	}

	for key, value := range outputs {
		_ = e.SetAttribute(ctx, flowID, stepID, key, value)
	}
}

func (e *Engine) executeStep(ctx context.Context, flowID, stepID timebox.ID) {
	stepCtx := e.PrepareStepExecution(ctx, flowID, stepID)
	if stepCtx == nil {
		return
	}

	stepCtx.execute(ctx)
}

// PrepareStepExecution validates and prepares a step for execution, returning
// an execution context or nil if preparation fails
func (e *Engine) PrepareStepExecution(
	ctx context.Context, flowID, stepID timebox.ID,
) *ExecContext {
	if err := e.validateStepExecution(ctx, flowID, stepID); err != nil {
		slog.Error("Step validation failed",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID),
			slog.Any("error", err))
		return nil
	}

	_, step, inputs := e.getStepExecutionData(ctx, flowID, stepID)
	if step == nil {
		return nil
	}

	if !e.shouldExecuteStep(ctx, flowID, stepID, step, inputs) {
		return nil
	}

	if err := e.StartStepExecution(ctx, flowID, stepID, inputs); err != nil {
		slog.Error("Failed to start step",
			slog.Any("error", err))
		return nil
	}

	return &ExecContext{
		engine: e,
		flowID: flowID,
		stepID: stepID,
		step:   step,
		inputs: inputs,
	}
}

func (e *Engine) getStepExecutionData(
	ctx context.Context, flowID, stepID timebox.ID,
) (*api.WorkflowState, *api.Step, api.Args) {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		slog.Error("Failed to get workflow state",
			slog.Any("error", err))
		return nil, nil, nil
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		slog.Error("Step not found",
			slog.Any("step_id", stepID))
		return nil, nil, nil
	}

	inputs := e.collectStepInputs(step, flow.GetAttributeArgs())
	return flow, step, inputs
}

func (e *Engine) shouldExecuteStep(
	ctx context.Context, flowID, stepID timebox.ID, step *api.Step,
	inputs api.Args,
) bool {
	if e.evaluateStepPredicate(ctx, flowID, stepID, step, inputs) {
		return true
	}

	reason := "predicate evaluated to false"
	slog.Info("Step skipped",
		slog.Any("step_id", stepID),
		slog.Any("flow_id", flowID),
		slog.String("reason", reason))

	if err := e.SkipStepExecution(ctx, flowID, stepID, reason); err != nil {
		slog.Error("Failed to skip step",
			slog.Any("error", err))
	}

	return false
}

func (e *Engine) evaluateStepPredicate(
	ctx context.Context, flowID, stepID timebox.ID, step *api.Step,
	inputs api.Args,
) bool {
	if step.Predicate == nil {
		return true
	}

	comp, err := e.GetCompiledPredicate(flowID, stepID)
	if err != nil {
		e.failPredicateEvaluation(ctx, flowID, stepID,
			"Failed to get compiled predicate",
			"predicate compilation failed", err)
		return false
	}

	if comp == nil {
		return true
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		e.failPredicateEvaluation(ctx, flowID, stepID,
			"Failed to get script environment for predicate",
			"failed to get script environment", err)
		return false
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		e.failPredicateEvaluation(ctx, flowID, stepID,
			"Failed to evaluate predicate",
			"predicate evaluation failed", err)
		return false
	}

	return shouldExecute
}

func (e *Engine) failPredicateEvaluation(
	ctx context.Context, flowID, stepID timebox.ID,
	logMsg, failMsg string, err error,
) {
	slog.Error(logMsg,
		slog.Any("step_id", stepID),
		slog.Any("error", err))

	if failErr := e.FailStepExecution(
		ctx, flowID, stepID,
		fmt.Sprintf("%s: %s", failMsg, err.Error()),
	); failErr != nil {
		slog.Error("Failed to record predicate failure",
			slog.Any("error", failErr))
	}
}

func (e *Engine) validateStepExecution(
	ctx context.Context, flowID, stepID timebox.ID,
) error {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return err
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return fmt.Errorf("%s: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
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
	e.start = time.Now()

	items := e.computeWorkItems()
	e.executeWorkItems(ctx, items)
}

// Work item execution functions

func (e *ExecContext) computeWorkItems() []api.Args {
	argNames := e.step.MultiArgNames()
	multiArgs := getMultiArgs(argNames, e.inputs)
	if len(multiArgs) == 0 {
		return []api.Args{e.inputs}
	}
	return cartesianProduct(multiArgs, e.inputs)
}

func (e *ExecContext) executeWorkItems(ctx context.Context, items []api.Args) {
	parallelism := 0
	if e.step.WorkConfig != nil {
		parallelism = e.step.WorkConfig.Parallelism
	}

	sem := make(chan struct{}, max(1, parallelism))

	for _, inputs := range items {
		go func(inputs api.Args) {
			sem <- struct{}{}
			defer func() { <-sem }()

			e.executeWorkItem(ctx, inputs)
		}(inputs)
	}
}

func (e *ExecContext) executeWorkItem(ctx context.Context, inputs api.Args) {
	if !e.engine.evaluateStepPredicate(
		ctx, e.flowID, e.stepID, e.step, inputs,
	) {
		return
	}

	token := api.Token(uuid.New().String())

	if err := e.engine.StartWork(
		ctx, e.flowID, e.stepID, token, inputs,
	); err != nil {
		return
	}

	outputs, err := e.performWork(ctx, inputs, token)

	if err != nil {
		e.handleWorkItemFailure(ctx, token, err)
		return
	}

	if !isAsyncStep(e.step.Type) {
		_ = e.engine.CompleteWork(ctx, e.flowID, e.stepID, token, outputs)
	}
}

func (e *ExecContext) handleWorkItemFailure(
	ctx context.Context, token api.Token, err error,
) {
	if failErr := e.engine.FailWork(
		ctx, e.flowID, e.stepID, token, err.Error(),
	); failErr != nil {
		return
	}

	flow, ferr := e.engine.GetWorkflowState(ctx, e.flowID)
	if ferr != nil {
		return
	}

	exec := flow.Executions[e.stepID]
	if exec == nil || exec.WorkItems == nil {
		return
	}

	workItem := exec.WorkItems[token]
	if workItem == nil {
		return
	}

	if e.engine.ShouldRetry(e.step, workItem) {
		_ = e.engine.ScheduleRetry(
			ctx, e.flowID, e.stepID, token, err.Error(),
		)
	}
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
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedStepType, e.step.Type)
	}
}

func (e *ExecContext) performScriptWork(
	_ context.Context, inputs api.Args,
) (api.Args, error) {
	c, err := e.engine.GetCompiledScript(e.flowID, e.stepID)
	if err != nil {
		return nil, fmt.Errorf("script compilation failed: %w", err)
	}

	if c == nil {
		return nil, ErrScriptCompileNil
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

func cartesianProduct(multiArgs MultiArgs, baseInputs api.Args) []api.Args {
	if len(multiArgs) == 0 {
		return nil
	}

	names, arrays := extractMultiArgs(multiArgs)

	var result []api.Args
	var generate func(int, api.Args)
	generate = func(depth int, current api.Args) {
		if depth == len(arrays) {
			result = append(result,
				combineInputs(baseInputs, current, multiArgs),
			)
			return
		}

		name := names[depth]
		for _, val := range arrays[depth] {
			next := current.Set(name, val)
			generate(depth+1, next)
		}
	}

	generate(0, api.Args{})
	return result
}

func extractMultiArgs(multiArgs MultiArgs) ([]api.Name, [][]any) {
	var names []api.Name
	var arrays [][]any
	for name, arr := range multiArgs {
		names = append(names, name)
		arrays = append(arrays, arr)
	}
	return names, arrays
}

func combineInputs(baseInputs, current api.Args, multiArgs MultiArgs) api.Args {
	inputs := api.Args{}
	for k, v := range baseInputs {
		if _, isMulti := multiArgs[k]; !isMulti {
			inputs[k] = v
		}
	}
	maps.Copy(inputs, current)
	return inputs
}
