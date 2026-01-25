package engine_test

import (
	"testing"
	"time"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestGetActiveFlow(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("active-test")

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"active-test"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-active-test", plan, api.Args{}, api.Metadata{})
		testify.NoError(t, err)

		flow, err := eng.GetFlowState("wf-active-test")
		testify.NoError(t, err)
		testify.NotNil(t, flow)
		testify.Equal(t, api.FlowID("wf-active-test"), flow.ID)
		testify.Equal(t, api.FlowActive, flow.Status)
	})
}

func TestGetFlowNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		_, err := eng.GetFlowState("nonexistent")
		testify.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestScript(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   `{:result "success"}`,
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"script-step"},
			Steps: api.Steps{step.ID: step},
		}

		consumer := env.EventHub.NewConsumer()
		err = env.Engine.StartFlow(
			"wf-script", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		helpers.WaitForFlowStarted(t,
			consumer, 5*time.Second, "wf-script",
		)

		fs := engine.FlowStep{FlowID: "wf-script", StepID: "script-step"}
		comp, err := env.Engine.GetCompiledScript(fs)
		testify.NoError(t, err)
		testify.NotNil(t, comp)
	})
}

func TestScriptMissing(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("no-script")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"no-script"},
			Steps: api.Steps{step.ID: step},
		}

		consumer := env.EventHub.NewConsumer()
		err = env.Engine.StartFlow(
			"wf-no-script", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		helpers.WaitForFlowStarted(t,
			consumer, 5*time.Second, "wf-no-script",
		)

		fs := engine.FlowStep{FlowID: "wf-no-script", StepID: "no-script"}
		comp, err := env.Engine.GetCompiledScript(fs)
		testify.NoError(t, err)
		testify.Nil(t, comp)
	})
}

func TestPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangLua, "return true",
		)

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{step.ID: step},
		}

		consumer := env.EventHub.NewConsumer()
		err = env.Engine.StartFlow(
			"wf-predicate", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		helpers.WaitForFlowStarted(t,
			consumer, 5*time.Second, "wf-predicate",
		)

		fs := engine.FlowStep{
			FlowID: "wf-predicate", StepID: "predicate-step",
		}
		comp, err := env.Engine.GetCompiledPredicate(fs)
		testify.NoError(t, err)
		testify.NotNil(t, comp)
	})
}

func TestPlanFlowNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		fs := engine.FlowStep{FlowID: "nonexistent-flow", StepID: "step-id"}
		_, err := eng.GetCompiledScript(fs)
		testify.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestStepMissingPlan(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("plan-step")

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			"wf-missing-plan-step", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		err = eng.FailStepExecution(
			engine.FlowStep{FlowID: "wf-missing-plan-step", StepID: "nope"}, "boom",
		)
		testify.ErrorIs(t, err, engine.ErrStepNotInPlan)
	})
}

func TestStepInvalidTransition(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("transition-step")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow(
			"wf-transition", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		env.WaitForStepStatus(t, "wf-transition", step.ID, 5*time.Second)

		err = env.Engine.FailStepExecution(
			engine.FlowStep{FlowID: "wf-transition", StepID: step.ID}, "late",
		)
		testify.ErrorIs(t, err, engine.ErrInvalidTransition)
	})
}
