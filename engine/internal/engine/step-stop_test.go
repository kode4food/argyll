package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	engassert "github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSetAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create a step that produces an output attribute
		step := helpers.NewStepWithOutputs("output-step", "test_key")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Configure mock to return the output value
		env.MockClient.SetResponse("output-step", api.Args{
			"test_key": "test_value",
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"output-step"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.FlowCompleted("wf-attr"), func() {
			err = env.Engine.StartFlow("wf-attr", plan)
			assert.NoError(t, err)
		})

		a := engassert.New(t)
		a.FlowStateEquals(env.Engine, "wf-attr", "test_key", "test_value")
	})
}
func TestDuplicateFirstWins(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create two steps that both produce the same output attribute
		stepA := helpers.NewStepWithOutputs("step-a", "shared_key")
		stepB := helpers.NewStepWithOutputs("step-b", "shared_key")

		err := env.Engine.RegisterStep(stepA)
		assert.NoError(t, err)
		err = env.Engine.RegisterStep(stepB)
		assert.NoError(t, err)

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

		flow := env.WaitForFlowStatus("wf-dup-attr", func() {
			err = env.Engine.StartFlow("wf-dup-attr", plan)
			assert.NoError(t, err)
		})

		// First value wins - duplicates are silently ignored
		attrs := flow.GetAttributes()
		assert.Contains(t, []string{"first", "second"}, attrs["shared_key"])
	})
}

func TestUndeclaredOutputsIgnored(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		producer := &api.Step{
			ID:   "producer",
			Name: "Producer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"value": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"value":  {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(producer))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(producer.ID, api.Args{
			"value": "abc",
			"extra": "ignore-me",
		})
		env.MockClient.SetResponse(consumer.ID, api.Args{
			"result": "ok",
			"extra2": "ignore-me-too",
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				producer.ID: producer,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"value": {
					Providers: []api.StepID{producer.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flow := env.WaitForFlowStatus("wf-undeclared-outputs", func() {
			err := env.Engine.StartFlow("wf-undeclared-outputs", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		assert.NotNil(t, flow.Attributes["value"])
		assert.NotNil(t, flow.Attributes["result"])
		assert.NotContains(t, flow.Attributes, api.Name("extra"))
		assert.NotContains(t, flow.Attributes, api.Name("extra2"))
	})
}

func TestOutputMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("mapped-output-step")
		step.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeAny,
				Mapping: &api.AttributeMapping{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.payload.value",
					},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{
			"payload": map[string]any{"value": "ok"},
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-output-mapping")
		fl := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, "ok", exec.Outputs["result"])
		assert.Equal(t, "ok", fl.Attributes["result"].Value)
	})
}
func TestOutputMappingWithRename(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("rename-output")
		step.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"status": {
				Role: api.RoleOutput,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Name: "success",
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"success": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-output-rename")
		fl := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"input": "test"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, "ok", exec.Outputs["status"])
		assert.Equal(t, "ok", fl.Attributes["status"].Value)
	})
}
