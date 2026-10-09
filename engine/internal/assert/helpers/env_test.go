package helpers_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type testTimer struct {
	ch chan time.Time
}

func TestEngine(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NotNil(t, env.Engine)
		assert.NotNil(t, env.MockClient)
		assert.NotNil(t, env.Config)
		assert.NotNil(t, env.EventHub)
		assert.NotNil(t, env.Cleanup)
	})
}

func TestEngineDependenciesClockOverride(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock: func() time.Time { return now },
	}, func(eng *engine.Engine) {
		assert.Equal(t, now, eng.Now())
	})
}

func TestEngineDependenciesTimerOverride(t *testing.T) {
	called := make(chan time.Duration, 1)
	makeTimer := func(delay time.Duration) scheduler.Timer {
		called <- delay
		return &testTimer{ch: make(chan time.Time)}
	}

	helpers.WithEngineDeps(t, engine.Dependencies{
		TimerConstructor: makeTimer,
	}, func(eng *engine.Engine) {
		assert.NoError(t, eng.Start())
		select {
		case delay := <-called:
			assert.Zero(t, delay)
		case <-time.After(time.Second):
			t.Fatal("timer constructor not called")
		}
	})
}

func TestEnvDepsPreserveDefaults(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time { return now },
	}, func(env *helpers.TestEngineEnv) {
		assert.NotNil(t, env.Engine)
		assert.NotNil(t, env.MockClient)
		assert.Equal(t, now, env.Engine.Now())
	})
}

func TestCanRegisterSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewTestStep()
		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
	})
}

func TestCanStartFlows(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewTestStep()
		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartPlan("test-wf", pl)
		assert.NoError(t, err)

		wf, err := eng.GetFlowState("test-wf")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("test-wf"), wf.ID)
	})
}

func TestCleanup(t *testing.T) {
	assert.NotPanics(t, func() {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NotNil(t, env.Engine)
		})
	})
}

func TestNewEngineInstance(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		eng, err := env.NewEngineInstance()
		assert.NoError(t, err)
		assert.NotNil(t, eng)
	})
}

func TestRaiseFlowEvents(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("raised-step")
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("raised-flow")

		err := env.RaiseFlowEvents(
			id,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: id,
					Plan:   pl,
					Init:   api.InitArgs{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowCompleted,
				Data: api.FlowCompletedEvent{
					FlowID: id,
					Result: api.Args{},
				},
			},
		)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestSeedFlow(t *testing.T) {
	for _, status := range []api.FlowStatus{
		api.FlowActive, api.FlowCompleted, api.FlowFailed,
	} {
		t.Run(string(status), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				tags := api.Tags{"tier:test"}
				err := env.SeedFlow("seeded-flow", status, tags)
				assert.NoError(t, err)

				fl, err := env.Engine.GetFlowState("seeded-flow")
				assert.NoError(t, err)
				assert.Equal(t, status, fl.Status)
				assert.Equal(t, tags, fl.Tags)
			})
		})
	}

	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		err := env.SeedFlow("invalid-flow", "invalid", nil)
		assert.True(t, errors.Is(err, helpers.ErrInvalidSeedFlowStatus))
	})
}

func (t *testTimer) Channel() <-chan time.Time {
	return t.ch
}

func (t *testTimer) Reset(time.Duration) bool {
	return true
}

func (t *testTimer) Stop() bool {
	return true
}
