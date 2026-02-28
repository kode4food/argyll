package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNoDeps(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{},
		},
	}

	assert.False(t, e.HasInputProvider("missing", flow))
}

func TestNoProviders(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{}},
			},
		},
	}

	assert.True(t, e.HasInputProvider("input", flow))
}

func TestCompletableProvider(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{"provider"}},
			},
			Steps: api.Steps{
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"input": {Role: api.RoleOutput, Type: api.TypeString},
					},
				},
			},
		},
		Executions: api.Executions{
			"provider": {Status: api.StepCompleted},
		},
	}

	assert.True(t, e.HasInputProvider("input", flow))
}

func TestFailedProvider(t *testing.T) {
	e := &engine.Engine{}
	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Attributes: api.AttributeGraph{
				"input": {Providers: []api.StepID{"provider"}},
			},
			Steps: api.Steps{
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"input": {Role: api.RoleOutput, Type: api.TypeString},
					},
				},
			},
		},
		Executions: api.Executions{
			"provider": {Status: api.StepFailed},
		},
	}

	assert.False(t, e.HasInputProvider("input", flow))
}

func TestGoalBlocked(t *testing.T) {
	e := &engine.Engine{}

	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Goals: []api.StepID{"goal"},
			Steps: api.Steps{
				"goal": {
					ID:   "goal",
					Name: "Goal",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"required": {
							Role: api.RoleRequired,
							Type: api.TypeString,
						},
					},
				},
				"provider": {
					ID:   "provider",
					Name: "Provider",
					Type: api.StepTypeSync,
					Attributes: api.AttributeSpecs{
						"required": {
							Role: api.RoleOutput,
							Type: api.TypeString,
						},
					},
				},
			},
			Attributes: api.AttributeGraph{
				"required": {Providers: []api.StepID{"provider"}},
			},
		},
		Executions: api.Executions{
			"goal": {Status: api.StepPending},
			// provider failed, so required input cannot be satisfied
			"provider": {Status: api.StepFailed},
		},
	}

	assert.True(t, e.IsFlowFailed(flow))
}

func TestGoalCompleted(t *testing.T) {
	e := &engine.Engine{}

	flow := &api.FlowState{
		Plan: &api.ExecutionPlan{
			Goals: []api.StepID{"goal"},
			Steps: api.Steps{
				"goal": {
					ID:   "goal",
					Name: "Goal",
					Type: api.StepTypeSync,
				},
			},
		},
		Executions: api.Executions{
			"goal": {Status: api.StepCompleted},
		},
	}

	assert.False(t, e.IsFlowFailed(flow))
}

func TestIsFlowFailed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		stepA := helpers.NewStepWithOutputs("step-a", "value")
		stepA.Attributes["value"].Type = api.TypeString

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
		plan.Attributes = api.AttributeGraph{}.AddStep(stepA).AddStep(stepB)

		err := env.Engine.RegisterStep(stepA)
		assert.NoError(t, err)
		err = env.Engine.RegisterStep(stepB)
		assert.NoError(t, err)
		env.MockClient.SetError(stepA.ID, errors.New("step failed"))

		env.WaitFor(wait.StepTerminal(api.FlowStep{
			FlowID: "wf-failed-test",
			StepID: stepA.ID,
		}), func() {
			err = env.Engine.StartFlow("wf-failed-test", plan)
			assert.NoError(t, err)
		})

		flow, err := env.Engine.GetFlowState("wf-failed-test")
		assert.NoError(t, err)
		assert.True(t, env.Engine.IsFlowFailed(flow))
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
		assert.NoError(t, err)

		err = eng.StartFlow("wf-ok-test", plan)
		assert.NoError(t, err)

		flow, err := eng.GetFlowState("wf-ok-test")
		assert.NoError(t, err)
		assert.False(t, eng.IsFlowFailed(flow))
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
		assert.NoError(t, err)
		err = eng.RegisterStep(stepB)
		assert.NoError(t, err)

		err = eng.StartFlow("wf-provider-test", plan)
		assert.NoError(t, err)

		flow, err := eng.GetFlowState("wf-provider-test")
		assert.NoError(t, err)
		assert.True(t, eng.HasInputProvider("value", flow))
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
		assert.NoError(t, err)

		err = eng.StartFlow("wf-no-provider-test", plan)
		assert.NoError(t, err)

		flow, err := eng.GetFlowState("wf-no-provider-test")
		assert.NoError(t, err)
		assert.False(t, eng.HasInputProvider("missing", flow))
	})
}

func TestGetStateNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		_, err := eng.GetFlowState("nonexistent")
		assert.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestGetFlowState(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("state-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"state-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-state", plan)
		assert.NoError(t, err)

		state, err := eng.GetFlowState("wf-state")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("wf-state"), state.ID)
		assert.NotNil(t, state.Status)
	})
}
func TestGetAttributes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create a step that produces multiple output attributes
		step := helpers.NewStepWithOutputs("step-attrs", "key1", "key2")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Configure mock to return multiple output values
		env.MockClient.SetResponse("step-attrs", api.Args{
			"key1": "value1",
			"key2": float64(42),
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-attrs"},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-getattrs", func() {
			err = env.Engine.StartFlow("wf-getattrs", plan)
			assert.NoError(t, err)
		})

		attrs := flow.GetAttributes()
		assert.Len(t, attrs, 2)
		assert.Equal(t, "value1", attrs["key1"])
		assert.Equal(t, float64(42), attrs["key2"])
	})
}

func TestGetAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("attr-step")
		step.Attributes = api.AttributeSpecs{
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
			Attributes: api.AttributeGraph{
				"result": {
					Providers: []api.StepID{step.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flow := env.WaitForFlowStatus("wf-attr", func() {
			err := env.Engine.StartFlow("wf-attr", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		value, ok, err := env.Engine.GetAttribute("wf-attr", "result")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "ok", value)

		_, ok, err = env.Engine.GetAttribute("wf-attr", "missing")
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}
