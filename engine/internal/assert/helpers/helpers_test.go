package helpers_test

import (
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

func TestMockClient(t *testing.T) {
	client := helpers.NewMockClient()
	assert.NotNil(t, client)
}

func TestSetResponse(t *testing.T) {
	cl := helpers.NewMockClient()

	out := api.Args{"result": "success"}
	cl.SetResponse("step-1", out)

	st := &api.Step{ID: "step-1"}
	result, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result["result"])
}

func TestSetError(t *testing.T) {
	cl := helpers.NewMockClient()

	expectedErr := assert.AnError
	cl.SetError("step-error", expectedErr)

	st := &api.Step{ID: "step-error"}
	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.Equal(t, expectedErr, err)
}

func TestTracksInvocations(t *testing.T) {
	cl := helpers.NewMockClient()

	step1 := &api.Step{ID: "step-1"}
	step2 := &api.Step{ID: "step-2"}

	_, _ = cl.Invoke(step1, api.Args{}, api.Metadata{})
	_, _ = cl.Invoke(step2, api.Args{}, api.Metadata{})

	assert.True(t, cl.WasInvoked("step-1"))
	assert.True(t, cl.WasInvoked("step-2"))
	assert.False(t, cl.WasInvoked("step-3"))

	invocations := cl.GetInvocations()
	assert.Len(t, invocations, 2)
	assert.Equal(t, api.StepID("step-1"), invocations[0])
	assert.Equal(t, api.StepID("step-2"), invocations[1])
}

func TestDefaultResponse(t *testing.T) {
	cl := helpers.NewMockClient()

	st := &api.Step{ID: "unconfigured-step"}
	result, err := cl.Invoke(st, api.Args{}, api.Metadata{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestThreadSafe(t *testing.T) {
	cl := helpers.NewMockClient()
	cl.SetResponse("step-1", api.Args{"result": "value"})

	done := make(chan bool)
	for range 10 {
		go func() {
			st := &api.Step{ID: "step-1"}
			_, _ = cl.Invoke(st, api.Args{}, api.Metadata{})
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	assert.True(t, cl.WasInvoked("step-1"))
	invocations := cl.GetInvocations()
	assert.Len(t, invocations, 10)
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
			assert.Equal(t, time.Duration(0), delay)
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
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartFlow("test-wf", pl)
		assert.NoError(t, err)

		wf, err := eng.GetFlowState("test-wf")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("test-wf"), wf.ID)
	})
}

func TestStep(t *testing.T) {
	st := helpers.NewTestStep()

	assert.NotNil(t, st)
	assert.NotEmpty(t, st.ID)
	assert.Equal(t, api.Name("Test Step"), st.Name)
	assert.Equal(t, api.StepTypeSync, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.NotEmpty(t, st.HTTP.Endpoint)

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepWithArgs(t *testing.T) {
	req := []api.Name{"req1", "req2"}
	opt := []api.Name{"opt1", "opt2"}

	st := helpers.NewTestStepWithArgs(req, opt)

	assert.NotNil(t, st)
	requiredArgs := st.GetRequiredArgs()
	optionalArgs := st.GetOptionalArgs()
	assert.Len(t, requiredArgs, 2)
	assert.Len(t, optionalArgs, 2)

	assert.Contains(t, requiredArgs, api.Name("req1"))
	assert.Contains(t, requiredArgs, api.Name("req2"))
	assert.Contains(t, optionalArgs, api.Name("opt1"))
	assert.Contains(t, optionalArgs, api.Name("opt2"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestConfig(t *testing.T) {
	cfg := helpers.NewTestConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "debug", cfg.LogLevel)

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestCleanup(t *testing.T) {
	assert.NotPanics(t, func() {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NotNil(t, env.Engine)
		})
	})
}

func TestSimpleStep(t *testing.T) {
	st := helpers.NewSimpleStep("test-id")

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("test-id"), st.ID)
	assert.Equal(t, api.StepTypeSync, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.Empty(t, st.GetRequiredArgs())
	assert.Empty(t, st.GetOptionalArgs())
	assert.Empty(t, st.GetOutputArgs())

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepWithOutput(t *testing.T) {
	st := helpers.NewStepWithOutputs("output-step", "result1", "result2")

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("output-step"), st.ID)
	outputArgs := st.GetOutputArgs()
	assert.Len(t, outputArgs, 2)
	assert.Contains(t, outputArgs, api.Name("result1"))
	assert.Contains(t, outputArgs, api.Name("result2"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestScriptStep(t *testing.T) {
	st := helpers.NewScriptStep(
		"script-id", api.ScriptLangAle, "{:result 42}", "result",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("script-id"), st.ID)
	assert.Equal(t, api.StepTypeScript, st.Type)
	assert.NotNil(t, st.Script)
	assert.Equal(t, api.ScriptLangAle, st.Script.Language)
	assert.Equal(t, "{:result 42}", st.Script.Script)
	assert.Len(t, st.GetOutputArgs(), 1)
	assert.Contains(t, st.GetOutputArgs(), api.Name("result"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestScriptNoOutput(t *testing.T) {
	st := helpers.NewScriptStep(
		"script-id", api.ScriptLangLua, "return {}",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepTypeScript, st.Type)
	assert.Empty(t, st.GetOutputArgs())
}

func TestStepPredicate(t *testing.T) {
	st := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangAle, "true", "output",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("pred-step"), st.ID)
	assert.Equal(t, api.StepTypeSync, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.NotNil(t, st.Predicate)
	assert.Equal(t, api.ScriptLangAle, st.Predicate.Language)
	assert.Equal(t, "true", st.Predicate.Script)
	assert.Len(t, st.GetOutputArgs(), 1)
	assert.Contains(t, st.GetOutputArgs(), api.Name("output"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepPredicateNoOutput(t *testing.T) {
	st := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangLua, "return false",
	)

	assert.NotNil(t, st)
	assert.NotNil(t, st.Predicate)
	assert.Empty(t, st.GetOutputArgs())
}

func TestLastMetadata(t *testing.T) {
	cl := helpers.NewMockClient()

	st := &api.Step{ID: "step-with-metadata"}
	md1 := api.Metadata{"attempt": "1"}
	md2 := api.Metadata{"attempt": "2"}
	md3 := api.Metadata{"attempt": "3"}

	_, _ = cl.Invoke(st, api.Args{}, md1)
	_, _ = cl.Invoke(st, api.Args{}, md2)
	_, _ = cl.Invoke(st, api.Args{}, md3)

	last := cl.LastMetadata("step-with-metadata")
	assert.NotNil(t, last)
	assert.Equal(t, "3", last["attempt"])
}

func TestMetadataEmpty(t *testing.T) {
	cl := helpers.NewMockClient()

	last := cl.LastMetadata("never-invoked")
	assert.Nil(t, last)
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
			Goals: []api.StepID{st.ID},
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
					Init:   api.Args{},
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

func (t *testTimer) Channel() <-chan time.Time {
	return t.ch
}

func (t *testTimer) Reset(time.Duration) bool {
	return true
}

func (t *testTimer) Stop() bool {
	return true
}
