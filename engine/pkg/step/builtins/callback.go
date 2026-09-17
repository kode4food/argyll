package builtins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

type (
	// CallbackEngine records the outcomes that step callbacks report
	CallbackEngine interface {
		GetFlowState(api.FlowID) (api.FlowState, error)
		CompleteWork(fs api.FlowStep, tkn api.Token, outputs api.Args) error
		FailWork(fs api.FlowStep, tkn api.Token, errMsg string) error
		CompleteCompensation(api.FlowStep, api.Token) error
		FailCompensation(fs api.FlowStep, tkn api.Token, errMsg string) error
	}

	// CallbackRequest carries the outcome an async step posts to the URL that a
	// CallbackURL built for it
	CallbackRequest struct {
		FlowStep api.FlowStep
		Token    api.Token
		Action   api.CallbackAction
		Request  *http.Request
	}
)

var (
	ErrInvalidCallback = errors.New("invalid callback")
)

// Callback records the outcome an async step reports through its callback URL
// and returns the HTTP status to answer with. Duplicates are accepted
func Callback(eng CallbackEngine, req *CallbackRequest) (int, error) {
	work, err := lookupWork(eng, req)
	if err != nil {
		return http.StatusBadRequest, err
	}
	switch req.Action {
	case api.ActionInvoke:
		return invokeCallback(eng, req)
	case api.ActionCompensate:
		return compensateCallback(eng, req, work.Status)
	default:
		return http.StatusBadRequest,
			fmt.Errorf("%w: action %s", ErrInvalidCallback, req.Action)
	}
}

func lookupWork(
	eng CallbackEngine, req *CallbackRequest,
) (api.WorkState, error) {
	fl, err := eng.GetFlowState(req.FlowStep.FlowID)
	if err != nil {
		return api.WorkState{}, fmt.Errorf("%w: %w", ErrInvalidCallback, err)
	}
	ex, ok := fl.Executions[req.FlowStep.StepID]
	if !ok {
		return api.WorkState{}, fmt.Errorf("%w: execution not found: %s",
			ErrInvalidCallback, req.FlowStep.StepID)
	}
	// Check if token matches a work item
	work, ok := ex.WorkItems[req.Token]
	if !ok {
		return api.WorkState{}, fmt.Errorf("%w: %w: %s",
			ErrInvalidCallback, api.ErrWorkItemNotFound, req.Token)
	}
	return work, nil
}

func invokeCallback(eng CallbackEngine, req *CallbackRequest) (int, error) {
	if isProblem(req) {
		var problem api.ProblemDetails
		if err := decodeBody(req, &problem); err != nil {
			return http.StatusBadRequest, err
		}
		return workStatus(
			eng.FailWork(req.FlowStep, req.Token, problem.Error()),
		)
	}
	var outputs api.Args
	if err := decodeBody(req, &outputs); err != nil {
		return http.StatusBadRequest, err
	}
	return workStatus(eng.CompleteWork(req.FlowStep, req.Token, outputs))
}

func compensateCallback(
	eng CallbackEngine, req *CallbackRequest, status api.WorkStatus,
) (int, error) {
	if !policy.WorkCompUnsettled(status) &&
		status != api.WorkCompensated &&
		status != api.WorkCompFailed {
		return http.StatusBadRequest,
			fmt.Errorf("%w: work item is not compensating: %s",
				ErrInvalidCallback, status)
	}
	if !isProblem(req) {
		return engineStatus(eng.CompleteCompensation(req.FlowStep, req.Token))
	}
	var problem api.ProblemDetails
	if err := decodeBody(req, &problem); err != nil {
		return http.StatusBadRequest, err
	}
	return engineStatus(
		eng.FailCompensation(req.FlowStep, req.Token, problem.Error()),
	)
}

func isProblem(req *CallbackRequest) bool {
	return api.IsProblemJSON(req.Request.Header.Get("Content-Type"))
}

func decodeBody(req *CallbackRequest, v any) error {
	if err := json.NewDecoder(req.Request.Body).Decode(v); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCallback, err)
	}
	return nil
}

func workStatus(err error) (int, error) {
	if errors.Is(err, api.ErrInvalidWorkTransition) {
		return http.StatusOK, nil
	}
	return engineStatus(err)
}

func engineStatus(err error) (int, error) {
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
