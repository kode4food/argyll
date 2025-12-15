package helpers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMockClient(t *testing.T) {
	client := helpers.NewMockClient()
	assert.NotNil(t, client)
}

func TestSetResponse(t *testing.T) {
	cl := helpers.NewMockClient()

	out := api.Args{"result": "success"}
	cl.SetResponse("step-1", out)

	step := &api.Step{ID: "step-1"}
	result, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)

	assert.NoError(t, err)
	assert.Equal(t, "success", result["result"])
}

func TestSetError(t *testing.T) {
	cl := helpers.NewMockClient()

	expectedErr := assert.AnError
	cl.SetError("step-error", expectedErr)

	step := &api.Step{ID: "step-error"}
	_, err := cl.Invoke(context.Background(), step, api.Args{}, api.Metadata{})

	assert.Equal(t, expectedErr, err)
}

func TestTracksInvocations(t *testing.T) {
	cl := helpers.NewMockClient()

	step1 := &api.Step{ID: "step-1"}
	step2 := &api.Step{ID: "step-2"}

	_, _ = cl.Invoke(context.Background(), step1, api.Args{}, api.Metadata{})
	_, _ = cl.Invoke(context.Background(), step2, api.Args{}, api.Metadata{})

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

	step := &api.Step{ID: "unconfigured-step"}
	result, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestThreadSafe(t *testing.T) {
	cl := helpers.NewMockClient()
	cl.SetResponse("step-1", api.Args{"result": "value"})

	done := make(chan bool)
	for range 10 {
		go func() {
			step := &api.Step{ID: "step-1"}
			_, _ = cl.Invoke(
				context.Background(), step, api.Args{}, api.Metadata{},
			)
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
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	assert.NotNil(t, env.Engine)
	assert.NotNil(t, env.Redis)
	assert.NotNil(t, env.MockClient)
	assert.NotNil(t, env.Config)
	assert.NotNil(t, env.EventHub)
	assert.NotNil(t, env.Cleanup)
}

func TestCanRegisterSteps(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewTestStep()
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	steps, err := env.Engine.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Len(t, steps, 1)
}

func TestCanStartFlows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewTestStep()
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "test-wf", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	wf, err := env.Engine.GetFlowState(context.Background(), "test-wf")
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("test-wf"), wf.ID)
}

func TestStep(t *testing.T) {
	step := helpers.NewTestStep()

	assert.NotNil(t, step)
	assert.NotEmpty(t, step.ID)
	assert.Equal(t, api.Name("Test Step"), step.Name)
	assert.Equal(t, api.StepTypeSync, step.Type)
	assert.NotNil(t, step.HTTP)
	assert.NotEmpty(t, step.HTTP.Endpoint)

	err := step.Validate()
	assert.NoError(t, err)
}

func TestStepWithArgs(t *testing.T) {
	req := []api.Name{"req1", "req2"}
	opt := []api.Name{"opt1", "opt2"}

	step := helpers.NewTestStepWithArgs(req, opt)

	assert.NotNil(t, step)
	requiredArgs := step.GetRequiredArgs()
	optionalArgs := step.GetOptionalArgs()
	assert.Len(t, requiredArgs, 2)
	assert.Len(t, optionalArgs, 2)

	assert.Contains(t, requiredArgs, api.Name("req1"))
	assert.Contains(t, requiredArgs, api.Name("req2"))
	assert.Contains(t, optionalArgs, api.Name("opt1"))
	assert.Contains(t, optionalArgs, api.Name("opt2"))

	err := step.Validate()
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
	env := helpers.NewTestEngine(t)

	// Verify resources exist
	assert.NotNil(t, env.Redis)
	assert.NotNil(t, env.Engine)

	// Cleanup should not panic
	assert.NotPanics(t, func() {
		env.Cleanup()
	})
}

func TestSimpleStep(t *testing.T) {
	step := helpers.NewSimpleStep("test-id")

	assert.NotNil(t, step)
	assert.Equal(t, api.StepID("test-id"), step.ID)
	assert.Equal(t, api.StepTypeSync, step.Type)
	assert.NotNil(t, step.HTTP)
	assert.Empty(t, step.GetRequiredArgs())
	assert.Empty(t, step.GetOptionalArgs())
	assert.Empty(t, step.GetOutputArgs())

	err := step.Validate()
	assert.NoError(t, err)
}

func TestStepWithOutput(t *testing.T) {
	step := helpers.NewStepWithOutputs("output-step", "result1", "result2")

	assert.NotNil(t, step)
	assert.Equal(t, api.StepID("output-step"), step.ID)
	outputArgs := step.GetOutputArgs()
	assert.Len(t, outputArgs, 2)
	assert.Contains(t, outputArgs, api.Name("result1"))
	assert.Contains(t, outputArgs, api.Name("result2"))

	err := step.Validate()
	assert.NoError(t, err)
}

func TestScriptStep(t *testing.T) {
	step := helpers.NewScriptStep(
		"script-id", api.ScriptLangAle, "{:result 42}", "result",
	)

	assert.NotNil(t, step)
	assert.Equal(t, api.StepID("script-id"), step.ID)
	assert.Equal(t, api.StepTypeScript, step.Type)
	assert.NotNil(t, step.Script)
	assert.Equal(t, api.ScriptLangAle, step.Script.Language)
	assert.Equal(t, "{:result 42}", step.Script.Script)
	assert.Len(t, step.GetOutputArgs(), 1)
	assert.Contains(t, step.GetOutputArgs(), api.Name("result"))

	err := step.Validate()
	assert.NoError(t, err)
}

func TestScriptNoOutput(t *testing.T) {
	step := helpers.NewScriptStep(
		"script-id", api.ScriptLangLua, "return {}",
	)

	assert.NotNil(t, step)
	assert.Equal(t, api.StepTypeScript, step.Type)
	assert.Empty(t, step.GetOutputArgs())
}

func TestStepPredicate(t *testing.T) {
	step := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangAle, "true", "output",
	)

	assert.NotNil(t, step)
	assert.Equal(t, api.StepID("pred-step"), step.ID)
	assert.Equal(t, api.StepTypeSync, step.Type)
	assert.NotNil(t, step.HTTP)
	assert.NotNil(t, step.Predicate)
	assert.Equal(t, api.ScriptLangAle, step.Predicate.Language)
	assert.Equal(t, "true", step.Predicate.Script)
	assert.Len(t, step.GetOutputArgs(), 1)
	assert.Contains(t, step.GetOutputArgs(), api.Name("output"))

	err := step.Validate()
	assert.NoError(t, err)
}

func TestStepPredicateNoOutput(t *testing.T) {
	step := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangLua, "return false",
	)

	assert.NotNil(t, step)
	assert.NotNil(t, step.Predicate)
	assert.Empty(t, step.GetOutputArgs())
}
