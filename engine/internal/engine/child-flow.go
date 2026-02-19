package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type parentWork struct {
	fs    api.FlowStep
	token api.Token
	step  *api.Step
}

var (
	ErrFlowOutputMissing = errors.New("flow output missing")
)

var (
	getMetaFlowID = api.GetMetaString[api.FlowID]
	getMetaStepID = api.GetMetaString[api.StepID]
	getMetaToken  = api.GetMetaString[api.Token]
)

func (e *Engine) StartChildFlow(
	parent api.FlowStep, token api.Token, step *api.Step, initState api.Args,
) (api.FlowID, error) {
	if step.Flow == nil || len(step.Flow.Goals) == 0 {
		return "", api.ErrFlowGoalsRequired
	}

	childID := childFlowID(parent, token)

	engState, err := e.GetEngineState()
	if err != nil {
		return "", err
	}

	plan, err := e.CreateExecutionPlan(engState, step.Flow.Goals, initState)
	if err != nil {
		return "", err
	}

	parentFlow, err := e.GetFlowState(parent.FlowID)
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

	if err := e.StartFlow(childID, plan,
		flowopt.WithInit(initState),
		flowopt.WithMetadata(meta),
	); err != nil {
		if errors.Is(err, ErrFlowExists) {
			return childID, nil
		}
		return "", err
	}

	return childID, nil
}

func (tx *flowTx) completeParentWork(st *api.FlowState) {
	target, ok := tx.parentWork(st)
	if !ok {
		return
	}
	switch st.Status {
	case api.FlowCompleted:
		childAttrs := st.GetAttributes()
		outputs, err := mapFlowOutputs(target.step, childAttrs)
		if err != nil {
			_ = tx.FailWork(target.fs, target.token, err.Error())
			return
		}
		_ = tx.CompleteWork(target.fs, target.token, outputs)
	case api.FlowFailed:
		errMsg := st.Error
		if errMsg == "" {
			errMsg = "child flow failed"
		}
		_ = tx.FailWork(target.fs, target.token, errMsg)
	}
}

func (tx *flowTx) parentWork(st *api.FlowState) (*parentWork, bool) {
	target := &parentWork{}
	if !tx.parentMeta(st, target) {
		return nil, false
	}

	parentFlow, err := tx.GetFlowState(target.fs.FlowID)
	if err != nil {
		return nil, false
	}

	exec := parentFlow.Executions[target.fs.StepID]
	workItem := exec.WorkItems[target.token]
	if isWorkTerminal(workItem.Status) {
		return nil, false
	}

	target.step = parentFlow.Plan.Steps[target.fs.StepID]
	return target, true
}

func (tx *flowTx) parentMeta(st *api.FlowState, target *parentWork) bool {
	if st.Metadata == nil {
		return false
	}

	flowID, hasFlowID := getMetaFlowID(st.Metadata, api.MetaParentFlowID)
	stepID, hasStepID := getMetaStepID(st.Metadata, api.MetaParentStepID)
	token, hasToken := getMetaToken(st.Metadata, api.MetaParentWorkItemToken)

	if !hasFlowID || !hasStepID || !hasToken {
		if !hasFlowID && !hasStepID && !hasToken {
			return false
		}
		slog.Error("Invalid parent metadata",
			log.FlowID(st.ID),
			slog.Bool("has_parent_flow_id", hasFlowID),
			slog.Bool("has_parent_step_id", hasStepID),
			slog.Bool("has_parent_token", hasToken))
		return false
	}

	target.fs = api.FlowStep{FlowID: flowID, StepID: stepID}
	target.token = token
	return true
}

func childFlowID(parent api.FlowStep, token api.Token) api.FlowID {
	return api.FlowID(
		fmt.Sprintf("%s:%s:%s", parent.FlowID, parent.StepID, token),
	)
}

func isWorkTerminal(status api.WorkStatus) bool {
	return status == api.WorkSucceeded || status == api.WorkFailed
}

func mapFlowOutputs(step *api.Step, childAttrs api.Args) (api.Args, error) {
	if step == nil {
		return nil, nil
	}

	outputs := maps.Clone(childAttrs)

	for _, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}
		if attr.Mapping == nil || attr.Mapping.Name == "" {
			continue
		}

		sourceName := api.Name(attr.Mapping.Name)
		value, ok := childAttrs[sourceName]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrFlowOutputMissing, sourceName)
		}
		outputs[sourceName] = value
	}

	return outputs, nil
}
