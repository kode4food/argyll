package engine_test

import (
	"context"
	"testing"
	"time"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestGetActiveFlow(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		ctx := context.Background()
		step := helpers.NewSimpleStep("active-test")

		err := eng.RegisterStep(ctx, step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"active-test"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			ctx, "wf-active-test", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		flow, err := eng.GetFlowState(ctx, "wf-active-test")
		testify.NoError(t, err)
		testify.NotNil(t, flow)
		testify.Equal(t, api.FlowID("wf-active-test"), flow.ID)
		testify.Equal(t, api.FlowActive, flow.Status)
	})
}

func TestGetFlowNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		ctx := context.Background()
		_, err := eng.GetFlowState(ctx, "nonexistent")
		testify.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestScript(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
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

		err := eng.RegisterStep(context.Background(), step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"script-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			context.Background(),
			"wf-script",
			plan,
			api.Args{},
			api.Metadata{},
		)
		testify.NoError(t, err)

		a := assert.New(t)
		fs := engine.FlowStep{FlowID: "wf-script", StepID: "script-step"}
		a.EventuallyWithError(func() error {
			_, err := eng.GetCompiledScript(fs)
			return err
		}, 500*time.Millisecond, "script should compile")

		comp, err := eng.GetCompiledScript(fs)
		testify.NoError(t, err)
		testify.NotNil(t, comp)
	})
}

func TestScriptMissing(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("no-script")

		err := eng.RegisterStep(context.Background(), step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"no-script"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			context.Background(),
			"wf-no-script",
			plan,
			api.Args{},
			api.Metadata{},
		)
		testify.NoError(t, err)

		fs := engine.FlowStep{FlowID: "wf-no-script", StepID: "no-script"}
		comp, err := eng.GetCompiledScript(fs)
		testify.NoError(t, err)
		testify.Nil(t, comp)
	})
}

func TestPredicate(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangLua, "return true",
		)

		err := eng.RegisterStep(context.Background(), step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			context.Background(),
			"wf-predicate",
			plan,
			api.Args{},
			api.Metadata{},
		)
		testify.NoError(t, err)

		a := assert.New(t)
		fs := engine.FlowStep{FlowID: "wf-predicate", StepID: "predicate-step"}
		a.EventuallyWithError(func() error {
			_, err := eng.GetCompiledPredicate(fs)
			return err
		}, 500*time.Millisecond, "predicate should compile")

		comp, err := eng.GetCompiledPredicate(fs)
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
		ctx := context.Background()
		step := helpers.NewSimpleStep("plan-step")

		err := eng.RegisterStep(ctx, step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow(
			ctx, "wf-missing-plan-step", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		err = eng.FailStepExecution(ctx,
			engine.FlowStep{FlowID: "wf-missing-plan-step", StepID: "nope"},
			"boom",
		)
		testify.ErrorIs(t, err, engine.ErrStepNotInPlan)
	})
}

func TestStepInvalidTransition(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		ctx := context.Background()
		step := helpers.NewSimpleStep("transition-step")

		err := env.Engine.RegisterStep(ctx, step)
		testify.NoError(t, err)
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow(
			ctx, "wf-transition", plan, api.Args{}, api.Metadata{},
		)
		testify.NoError(t, err)

		env.WaitForStepStatus(t, ctx, "wf-transition", step.ID, 5*time.Second)

		err = env.Engine.FailStepExecution(ctx,
			engine.FlowStep{FlowID: "wf-transition", StepID: step.ID},
			"late",
		)
		testify.ErrorIs(t, err, engine.ErrInvalidTransition)
	})
}
