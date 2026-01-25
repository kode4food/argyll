package engine_test

import (
	"errors"
	"testing"
	"time"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const testTimeout = 5 * time.Second

func TestStartDuplicate(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-1")

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-dup", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		err = eng.StartFlow("wf-dup", plan, api.Args{}, api.Metadata{})
		testify.ErrorIs(t, err, engine.ErrFlowExists)
	})
}

func TestStartFlowSchedulesInitialWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := helpers.NewSimpleStep("step-start")
		step.Type = api.StepTypeAsync
		step.HTTP.Timeout = 30 * api.Second

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-start", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := env.Engine.GetFlowState("wf-start")
		testify.NoError(t, err)

		exec := flow.Executions[step.ID]
		testify.Equal(t, api.StepActive, exec.Status)
		testify.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			testify.Equal(t, api.WorkActive, item.Status)
		}
	})
}

func TestStartMissingInput(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-needs-input")
		step.Attributes["required_value"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{"step-needs-input"},
			Steps:    api.Steps{step.ID: step},
			Required: []api.Name{"required_value"},
		}

		err := eng.StartFlow("wf-missing", plan, api.Args{}, api.Metadata{})
		testify.Error(t, err)
	})
}

func TestGetStateNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		_, err := eng.GetFlowState("nonexistent")
		testify.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestSetAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		// Create a step that produces an output attribute
		step := helpers.NewStepWithOutputs("output-step", "test_key")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		// Configure mock to return the output value
		env.MockClient.SetResponse("output-step", api.Args{
			"test_key": "test_value",
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"output-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-attr", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for flow to complete
		env.WaitForFlowStatus(t, "wf-attr", testTimeout)

		a := assert.New(t)
		a.FlowStateEquals(env.Engine, "wf-attr", "test_key", "test_value")
	})
}

func TestGetAttributes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		// Create a step that produces multiple output attributes
		step := helpers.NewStepWithOutputs("step-attrs", "key1", "key2")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		// Configure mock to return multiple output values
		env.MockClient.SetResponse("step-attrs", api.Args{
			"key1": "value1",
			"key2": float64(42),
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-attrs"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-getattrs", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for flow to complete
		flow := env.WaitForFlowStatus(t, "wf-getattrs", testTimeout)

		attrs := flow.GetAttributes()
		testify.Len(t, attrs, 2)
		testify.Equal(t, "value1", attrs["key1"])
		testify.Equal(t, float64(42), attrs["key2"])
	})
}

func TestDuplicateFirstWins(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		// Create two steps that both produce the same output attribute
		stepA := helpers.NewStepWithOutputs("step-a", "shared_key")
		stepB := helpers.NewStepWithOutputs("step-b", "shared_key")

		err := env.Engine.RegisterStep(stepA)
		testify.NoError(t, err)
		err = env.Engine.RegisterStep(stepB)
		testify.NoError(t, err)

		// Configure mock responses - step-a runs first and sets "first"
		env.MockClient.SetResponse("step-a", api.Args{"shared_key": "first"})
		env.MockClient.SetResponse("step-b", api.Args{"shared_key": "second"})

		// Both steps are goals so both will execute
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-a", "step-b"},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}

		err = env.Engine.StartFlow("wf-dup-attr", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for flow to complete
		flow := env.WaitForFlowStatus(t, "wf-dup-attr", testTimeout)

		// First value wins - duplicates are silently ignored
		attrs := flow.GetAttributes()
		testify.Contains(t, []string{"first", "second"}, attrs["shared_key"])
	})
}

func TestCompleteFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		env.Engine.Start()

		step := helpers.NewStepWithOutputs("complete-step", "result")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		// Configure mock to return a result
		env.MockClient.SetResponse("complete-step", api.Args{"result": "final"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"complete-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-complete", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for flow to complete automatically
		flow := env.WaitForFlowStatus(t, "wf-complete", testTimeout)
		a.FlowStatus(flow, api.FlowCompleted)
	})
}

func TestFailFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		env.Engine.Start()

		step := helpers.NewSimpleStep("fail-step")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		env.MockClient.SetError("fail-step", errors.New("test error"))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"fail-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-fail", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for flow to fail automatically
		flow := env.WaitForFlowStatus(t, "wf-fail", testTimeout)
		a.FlowStatus(flow, api.FlowFailed)
		testify.Contains(t, flow.Error, "test error")
	})
}

func TestSkipStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		// Create a step with a predicate that returns false, causing a  skip
		step := helpers.NewStepWithPredicate(
			"step-skip", api.ScriptLangAle, "false",
		)

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-skip"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-skip", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Wait for step to be skipped
		exec := env.WaitForStepStatus(t, "wf-skip", "step-skip", testTimeout)
		testify.NotNil(t, exec)
		testify.Equal(t, api.StepSkipped, exec.Status)
		testify.Equal(t, "predicate returned false", exec.Error)
	})
}

func TestStartFlowSimple(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := &api.Step{
			ID:   "goal-step",
			Name: "Goal",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{"goal-step"},
			Required: []api.Name{},
			Steps: api.Steps{
				"goal-step": step,
			},
		}

		err = env.Engine.StartFlow("wf-simple", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := env.Engine.GetFlowState("wf-simple")
		testify.NoError(t, err)
		testify.NotNil(t, flow)
		testify.Equal(t, api.FlowID("wf-simple"), flow.ID)
	})
}

func TestListFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewSimpleStep("test")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"test"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-list", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		helpers.WaitForFlowActivated(
			t, env.EventHub, []api.FlowID{"wf-list"}, testTimeout,
		)

		flows, err := env.Engine.ListFlows()
		testify.NoError(t, err)
		testify.NotEmpty(t, flows)
	})
}

func TestIsFlowFailed(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := helpers.NewStepWithOutputs("step-a", "value")

		stepB := helpers.NewSimpleStep("step-b")
		stepB.Attributes["value"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-b"},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}

		err := eng.RegisterStep(stepA)
		testify.NoError(t, err)
		err = eng.RegisterStep(stepB)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-failed-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		// Fail step-a directly via FailStepExecution (step will be started)
		fs := engine.FlowStep{FlowID: "wf-failed-test", StepID: "step-a"}
		err = eng.FailStepExecution(fs, "test error")
		testify.NoError(t, err)

		flow, err := eng.GetFlowState("wf-failed-test")
		testify.NoError(t, err)

		isFailed := eng.IsFlowFailed(flow)
		testify.True(t, isFailed)
	})
}

func TestIsFlowNotFailed(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-ok")

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-ok"},
			Steps: api.Steps{step.ID: step},
		}

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-ok-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := eng.GetFlowState("wf-ok-test")
		testify.NoError(t, err)

		isFailed := eng.IsFlowFailed(flow)
		testify.False(t, isFailed)
	})
}

func TestHasInputProvider(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := helpers.NewStepWithOutputs("step-a", "value")

		stepB := helpers.NewSimpleStep("step-b")
		stepB.Attributes["value"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-b"},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
			Attributes: api.AttributeGraph{
				"value": {
					Providers: []api.StepID{stepA.ID},
					Consumers: []api.StepID{stepB.ID},
				},
			},
		}

		err := eng.RegisterStep(stepA)
		testify.NoError(t, err)
		err = eng.RegisterStep(stepB)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-provider-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := eng.GetFlowState("wf-provider-test")
		testify.NoError(t, err)

		hasProvider := eng.HasInputProvider("value", flow)
		testify.True(t, hasProvider)
	})
}

func TestHasInputProviderNone(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-alone")
		step.Attributes["missing"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		plan := &api.ExecutionPlan{
			Goals:      []api.StepID{"step-alone"},
			Steps:      api.Steps{step.ID: step},
			Attributes: api.AttributeGraph{},
		}

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-no-provider-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := eng.GetFlowState("wf-no-provider-test")
		testify.NoError(t, err)

		hasProvider := eng.HasInputProvider("missing", flow)
		testify.False(t, hasProvider)
	})
}

func TestStepProvidesInput(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := helpers.NewStepWithOutputs("step-provider", "result")

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-provider"},
			Steps: api.Steps{
				stepA.ID: stepA,
			},
		}

		err := eng.RegisterStep(stepA)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-provides-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		outputArgs := stepA.GetOutputArgs()
		testify.Contains(t, outputArgs, api.Name("result"))
		testify.NotContains(t, outputArgs, api.Name("other"))
	})
}
