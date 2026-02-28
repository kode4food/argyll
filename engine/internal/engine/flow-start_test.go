package engine_test

import (
	"testing"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStartDuplicate(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-1")

		err := eng.RegisterStep(step)
		testify.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-dup", plan)
		testify.NoError(t, err)

		err = eng.StartFlow("wf-dup", plan)
		testify.ErrorIs(t, err, engine.ErrFlowExists)
	})
}

func TestStartFlowSchedulesWork(t *testing.T) {
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

		err = env.Engine.StartFlow("wf-start", plan)
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

		err := eng.StartFlow("wf-missing", plan)
		testify.Error(t, err)
	})
}

func TestStartFlowRejectsPartialParentMetadata(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("step-parent-meta")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := eng.StartFlow("wf-partial-parent-meta", plan,
			flowopt.WithMetadata(api.Metadata{
				api.MetaParentFlowID: "parent",
			}),
		)
		testify.ErrorContains(t, err, "partial parent metadata")
	})
}

func TestStartFlowSimple(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		err = env.Engine.StartFlow("wf-simple", plan)
		testify.NoError(t, err)

		flow, err := env.Engine.GetFlowState("wf-simple")
		testify.NoError(t, err)
		testify.NotNil(t, flow)
		testify.Equal(t, api.FlowID("wf-simple"), flow.ID)
	})
}
