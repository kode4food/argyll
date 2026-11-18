package builtins

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

type (
	// CallbackEngine records the outcomes that step callbacks report
	CallbackEngine interface {
		GetFlowState(api.FlowID) (api.FlowState, error)
		CompleteWork(fs api.FlowStep, tkn api.Token, outputs api.Args, usage api.Usage) error
		RetryWork(fs api.FlowStep, tkn api.Token, cause error, usage api.Usage) error
		FailWork(fs api.FlowStep, tkn api.Token, cause error, usage api.Usage) error
		CompleteCompensation(api.FlowStep, api.Token) error
		FailCompensation(fs api.FlowStep, tkn api.Token, errMsg string) error
	}

	// CallbackRequest carries the outcome an async step reported for one of its work items: its
	// outputs, or the problem that ended it, with the usage the attempt consumed
	CallbackRequest struct {
		FlowStep api.FlowStep
		Token    api.Token
		Action   api.CallbackAction
		Outputs  api.Args
		Problem  *api.ProblemDetails
		Usage    api.Usage
	}
)

var (
	ErrInvalidCallback           = errors.New("invalid callback")
	ErrInvalidCallbackAction     = errors.New("unsupported action")
	ErrCallbackExecutionNotFound = errors.New("execution not found")
	ErrCallbackNotCompensating   = errors.New("work item is not compensating")
)

// Callback records the outcome an async step reported. An outcome its work item already settled is
// accepted as a duplicate
func Callback(eng CallbackEngine, req *CallbackRequest) error {
	work, err := lookupWork(eng, req)
	if err != nil {
		return err
	}
	switch req.Action {
	case api.ActionInvoke, api.ActionAgent:
		return ignoreDuplicate(invokeCallback(eng, req))
	case api.ActionCompensate:
		return compensateCallback(eng, req, work.Status)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCallbackAction, req.Action)
	}
}

func lookupWork(eng CallbackEngine, req *CallbackRequest) (api.WorkState, error) {
	fl, err := eng.GetFlowState(req.FlowStep.FlowID)
	if err != nil {
		return api.WorkState{}, fmt.Errorf("%w: %v", ErrInvalidCallback, err)
	}
	sid := req.FlowStep.StepID
	ex, ok := fl.Executions[sid]
	if !ok {
		return api.WorkState{}, fmt.Errorf("%w: %s", ErrCallbackExecutionNotFound, sid)
	}
	// Check if token matches a work item
	work, ok := ex.WorkItems[req.Token]
	if !ok {
		return api.WorkState{}, fmt.Errorf("%w: %s", api.ErrWorkItemNotFound, req.Token)
	}
	return work, nil
}

// invokeCallback settles the work item: a problem retries when its status is retryable and fails
// otherwise
func invokeCallback(eng CallbackEngine, req *CallbackRequest) error {
	switch {
	case req.Problem == nil:
		return eng.CompleteWork(req.FlowStep, req.Token, req.Outputs, req.Usage)
	case retryableHTTPStatus(req.Problem.Status):
		return eng.RetryWork(req.FlowStep, req.Token, req.Problem, req.Usage)
	default:
		return eng.FailWork(req.FlowStep, req.Token, req.Problem, req.Usage)
	}
}

func compensateCallback(eng CallbackEngine, req *CallbackRequest, status api.WorkStatus) error {
	if !policy.WorkCompUnsettled(status) &&
		status != api.WorkCompensated &&
		status != api.WorkCompFailed {
		return fmt.Errorf("%w: %s", ErrCallbackNotCompensating, status)
	}
	if req.Problem == nil {
		return eng.CompleteCompensation(req.FlowStep, req.Token)
	}
	return eng.FailCompensation(req.FlowStep, req.Token, req.Problem.Error())
}

// ignoreDuplicate accepts an outcome for a work item that has already settled
func ignoreDuplicate(err error) error {
	if errors.Is(err, api.ErrInvalidWorkTransition) {
		return nil
	}
	return err
}
