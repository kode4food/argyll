package engine

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	getMetaFlowID = api.GetMetaString[api.FlowID]
	getMetaStepID = api.GetMetaString[api.StepID]
	getMetaToken  = api.GetMetaString[api.Token]
)

var ErrFlowOutputMissing = errors.New("flow output missing")

func (e *Engine) StartChildFlow(
	ctx context.Context, parent FlowStep, token api.Token, step *api.Step,
	initState api.Args,
) (api.FlowID, error) {
	if step.Flow == nil || len(step.Flow.Goals) == 0 {
		return "", api.ErrFlowGoalsRequired
	}

	childID := childFlowID(parent, token)

	engState, err := e.GetEngineState(ctx)
	if err != nil {
		return "", err
	}

	plan, err := e.CreateExecutionPlan(engState, step.Flow.Goals, initState)
	if err != nil {
		return "", err
	}

	parentFlow, err := e.GetFlowState(ctx, parent.FlowID)
	if err != nil {
		return "", err
	}

	meta := maps.Clone(parentFlow.Metadata)
	if meta == nil {
		meta = api.Metadata{}
	}
	meta[api.MetaParentFlowID] = parent.FlowID
	meta[api.MetaParentStepID] = parent.StepID
	meta[api.MetaParentWorkItemToken] = token

	if err := e.StartFlow(ctx, childID, plan, initState, meta); err != nil {
		if errors.Is(err, ErrFlowExists) {
			return childID, nil
		}
		return "", err
	}

	return childID, nil
}

func (a *flowActor) completeParentWork(flow *api.FlowState) {
	if flow.Metadata == nil {
		return
	}

	parentFlowID, ok := getMetaFlowID(flow.Metadata, api.MetaParentFlowID)
	if !ok {
		return
	}
	parentStepID, ok := getMetaStepID(flow.Metadata, api.MetaParentStepID)
	if !ok {
		return
	}
	parentToken, ok := getMetaToken(
		flow.Metadata, api.MetaParentWorkItemToken,
	)
	if !ok {
		return
	}

	parentFlow, err := a.GetFlowState(context.Background(), parentFlowID)
	if err != nil {
		return
	}

	exec, ok := parentFlow.Executions[parentStepID]
	if !ok || exec.WorkItems == nil {
		return
	}

	workItem, ok := exec.WorkItems[parentToken]
	if !ok || workItem == nil {
		return
	}

	if isWorkTerminal(workItem.Status) {
		return
	}

	step := getPlanStep(parentFlow, parentStepID)
	if step == nil {
		return
	}

	fs := FlowStep{FlowID: parentFlowID, StepID: parentStepID}
	switch flow.Status {
	case api.FlowCompleted:
		childAttrs := flow.GetAttributes()
		outputs, err := mapFlowOutputs(step, childAttrs)
		if err != nil {
			_ = a.FailWork(context.Background(), fs, parentToken, err.Error())
			return
		}
		_ = a.CompleteWork(context.Background(), fs, parentToken, outputs)
	case api.FlowFailed:
		errMsg := flow.Error
		if errMsg == "" {
			errMsg = "child flow failed"
		}
		_ = a.FailWork(context.Background(), fs, parentToken, errMsg)
	}
}

func childFlowID(parent FlowStep, token api.Token) api.FlowID {
	return api.FlowID(
		fmt.Sprintf("%s:%s:%s", parent.FlowID, parent.StepID, token),
	)
}

func isWorkTerminal(status api.WorkStatus) bool {
	return status == api.WorkSucceeded || status == api.WorkFailed
}

func mapFlowInputs(step *api.Step, inputs api.Args) api.Args {
	if step.Flow == nil || len(step.Flow.InputMap) == 0 {
		return inputs
	}

	mapped := api.Args{}
	for name, value := range inputs {
		target := name
		if mappedName, ok := step.Flow.InputMap[name]; ok {
			target = mappedName
		}
		mapped[target] = value
	}

	return mapped
}

func mapFlowOutputs(step *api.Step, childAttrs api.Args) (api.Args, error) {
	if step.Flow == nil {
		return nil, nil
	}

	outputs := api.Args{}
	outputSet := map[api.Name]struct{}{}
	for _, name := range step.GetOutputArgs() {
		outputSet[name] = struct{}{}
	}

	for childName, outputName := range step.Flow.OutputMap {
		if _, ok := outputSet[outputName]; !ok {
			continue
		}
		value, ok := childAttrs[childName]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrFlowOutputMissing, childName)
		}
		outputs[outputName] = value
	}

	for outputName := range outputSet {
		if _, ok := outputs[outputName]; ok {
			continue
		}
		if value, ok := childAttrs[outputName]; ok {
			outputs[outputName] = value
		}
	}

	return outputs, nil
}

func getPlanStep(flow *api.FlowState, stepID api.StepID) *api.Step {
	if flow.Plan == nil {
		return nil
	}
	if flow.Plan.Steps == nil {
		return nil
	}
	return flow.Plan.Steps[stepID]
}
