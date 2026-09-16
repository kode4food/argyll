package argyll_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/argyll"
	"github.com/kode4food/argyll/engine/pkg/step"
)

const (
	compensationWait = 5 * time.Second
	compensationPoll = 10 * time.Millisecond
)

func TestLocalCompensation(t *testing.T) {
	tests := []struct {
		name  string
		async bool
		retry bool
	}{
		{name: "sync"},
		{name: "async", async: true},
		{name: "retry", retry: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := make(chan step.CompensateRequest, 1)
			var attempts atomic.Int32
			eng := newTestEngine(t, argyll.Options{
				Handlers: step.Handlers{
					localStepType: {
						Execute: func(
							rt step.Runtime, st *api.Step,
							_ api.Args, token api.Token,
						) error {
							if st.ID == "fail" {
								return errors.New("downstream failed")
							}
							return rt.CompleteWork(token, api.Args{
								"value": "done",
							})
						},
						Compensate: func(
							req step.CompensateRequest,
						) (bool, error) {
							if attempts.Add(1) == 1 && test.retry {
								return false, api.ErrWorkNotCompleted
							}
							requests <- req
							return !test.async, nil
						},
					},
				},
			})
			assert.NoError(t, eng.Start())
			assert.NoError(t, eng.RegisterStep(&api.Step{
				ID:       "local",
				Name:     "Local",
				Type:     localStepType,
				Handling: api.HandlingCompensated,
				Attributes: api.AttributeSpecs{
					"value": {Role: api.RoleOutput, Compensated: true},
				},
			}))
			assert.NoError(t, eng.RegisterStep(&api.Step{
				ID:   "fail",
				Name: "Fail",
				Type: localStepType,
				Attributes: api.AttributeSpecs{
					"value": {Role: api.RoleRequired},
				},
			}))
			assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
				ID:         "rollback",
				Goals:      []api.StepID{"fail"},
				Compensate: true,
			}))
			var req step.CompensateRequest
			select {
			case req = <-requests:
			case <-time.After(compensationWait):
				t.Fatal("compensation was never executed")
			}
			assert.Equal(t, api.FlowID("rollback"), req.FlowID)
			assert.Equal(t, api.Args{"value": "done"}, req.Outputs)
			assert.Nil(t, req.Step.HTTP)
			assert.Equal(t, req.Token, req.Metadata[api.MetaReceiptToken])
			if test.async {
				fl, err := eng.GetFlowState(req.FlowID)
				assert.NoError(t, err)
				assert.Equal(t, api.WorkCompensating,
					fl.Executions[req.Step.ID].WorkItems[req.Token].Status)
				assert.NoError(t, eng.CompleteCompensation(api.FlowStep{
					FlowID: req.FlowID,
					StepID: req.Step.ID,
				}, req.Token))
			}
			assert.Eventually(t, func() bool {
				fl, err := eng.GetFlowState(req.FlowID)
				return err == nil &&
					fl.Executions[req.Step.ID].WorkItems[req.Token].Status ==
						api.WorkCompensated
			}, compensationWait, compensationPoll)
			if test.retry {
				assert.Equal(t, int32(2), attempts.Load())
			}
		})
	}
}

func TestMissingCompensator(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{
		Handlers: step.Handlers{localStepType: {}},
	})
	err := eng.RegisterStep(&api.Step{
		ID:       "local",
		Name:     "Local",
		Type:     localStepType,
		Handling: api.HandlingCompensated,
	})
	assert.ErrorIs(t, err, api.ErrInvalidStep)
	assert.ErrorIs(t, err, step.ErrCompensatorRequired)
}
