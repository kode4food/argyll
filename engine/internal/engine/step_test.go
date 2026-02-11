package engine_test

import (
	"testing"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
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

		err = eng.StartFlow("wf-active-test", plan)
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

		env.WaitFor(wait.FlowStarted("wf-script"), func() {
			err = env.Engine.StartFlow("wf-script", plan)
			testify.NoError(t, err)
		})

		fs := api.FlowStep{FlowID: "wf-script", StepID: "script-step"}
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

		env.WaitFor(wait.FlowStarted("wf-no-script"), func() {
			err = env.Engine.StartFlow("wf-no-script", plan)
			testify.NoError(t, err)
		})

		fs := api.FlowStep{FlowID: "wf-no-script", StepID: "no-script"}
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

		env.WaitFor(wait.FlowStarted("wf-predicate"), func() {
			err = env.Engine.StartFlow("wf-predicate", plan)
			testify.NoError(t, err)
		})

		fs := api.FlowStep{
			FlowID: "wf-predicate", StepID: "predicate-step",
		}
		comp, err := env.Engine.GetCompiledPredicate(fs)
		testify.NoError(t, err)
		testify.NotNil(t, comp)
	})
}

func TestPlanFlowNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		fs := api.FlowStep{FlowID: "nonexistent-flow", StepID: "step-id"}
		_, err := eng.GetCompiledScript(fs)
		testify.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}
