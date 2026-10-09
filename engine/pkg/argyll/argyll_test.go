package argyll_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/argyll"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

const (
	localStepType api.StepType = "local"

	compensationWait = 5 * time.Second
	compensationPoll = 10 * time.Millisecond
)

func TestEmbeddedFlow(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{})
	assert.NoError(t, eng.Start())

	st := &api.Step{
		ID:   "embedded-step",
		Name: "Embedded Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return { greeting = "hello" }`,
		},
		Attributes: api.AttributeSpecs{
			"greeting": {Role: api.RoleOutput},
		},
	}
	assert.NoError(t, eng.RegisterStep(st))

	steps, err := eng.ListSteps()
	assert.NoError(t, err)
	assert.Len(t, steps, 1)

	assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
		ID:    "embedded-flow",
		Goals: api.Goals{Steps: []api.StepID{st.ID}},
	}))

	fl, err := eng.GetFlowState("embedded-flow")
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("embedded-flow"), fl.ID)
}

func TestEmbeddedFlowUnknownGoal(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{})

	err := eng.StartFlow(api.CreateFlowRequest{
		ID:    "embedded-flow",
		Goals: api.Goals{Steps: []api.StepID{"nope"}},
	})
	assert.ErrorIs(t, err, api.ErrGoalNotFound)
}

func TestEmbeddedStepType(t *testing.T) {
	ran := make(chan api.StepID, 1)
	eng := newTestEngine(t, argyll.Options{
		Handlers: step.Handlers{
			api.StepTypeScript: builtins.Script(),
			localStepType: {
				Invoke: func(
					rt step.Runtime, st *api.Step,
					inputs api.Args, tkn api.Token,
				) error {
					ran <- st.ID
					return rt.CompleteWork(tkn, api.Args{
						"greeting": "hello",
					})
				},
			},
		},
	})
	assert.NoError(t, eng.Start())

	st := &api.Step{
		ID:   "local-step",
		Name: "Local Step",
		Type: localStepType,
		Attributes: api.AttributeSpecs{
			"greeting": {Role: api.RoleOutput},
		},
	}
	assert.NoError(t, eng.RegisterStep(st))
	assert.NoError(t, eng.RegisterStep(&api.Step{
		ID:   "script-step",
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {}",
		},
	}))
	assert.ErrorIs(t, eng.RegisterStep(&api.Step{
		ID:   "http-step",
		Name: "HTTP Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.test"},
		},
	}), api.ErrInvalidStepType)
	assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
		ID:    "local-flow",
		Goals: api.Goals{Steps: []api.StepID{st.ID}},
	}))

	select {
	case sid := <-ran:
		assert.Equal(t, st.ID, sid)
	case <-time.After(5 * time.Second):
		t.Fatal("handler was never executed")
	}

	assert.Eventually(t, func() bool {
		fl, err := eng.GetFlowState("local-flow")
		return err == nil && fl.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)
}

func TestConfiguredHandlersAreExact(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{Handlers: step.Handlers{}})
	err := eng.RegisterStep(&api.Step{
		ID:   "script-step",
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {}",
		},
	})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

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
						Invoke: func(
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
				Goals:      api.Goals{Steps: []api.StepID{"fail"}},
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

func newTestEngine(t *testing.T, opts argyll.Options) argyll.Engine {
	t.Helper()

	opts.Backend = func(pub timebox.Publisher) (timebox.Backend, error) {
		return memory.Open(memory.Config{Publisher: pub}), nil
	}

	eng, err := argyll.New(opts)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = eng.Stop() })
	return eng
}
