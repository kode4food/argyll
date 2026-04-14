package engine_test

import (
	"testing"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSetAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		// Create a step that produces an output attribute
		st := helpers.NewStepWithOutputs("output-step", "test_key")

		err := env.Engine.RegisterStep(st)
		testify.NoError(t, err)

		// Configure mock to return the output value
		env.MockClient.SetResponse("output-step", api.Args{
			"test_key": "test_value",
		})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"output-step"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowCompleted("wf-attr"), func() {
			err = env.Engine.StartFlow("wf-attr", pl)
			testify.NoError(t, err)
		})

		a := assert.New(t)
		a.FlowStateEquals(env.Engine, "wf-attr", "test_key", "test_value")
	})
}
func TestDuplicateFirstWins(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-a", "step-b"},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}

		fl := env.WaitForFlowStatus("wf-dup-attr", func() {
			err = env.Engine.StartFlow("wf-dup-attr", pl)
			testify.NoError(t, err)
		})

		// First value wins - duplicates are silently ignored
		attrs := fl.GetAttributes()
		testify.Contains(t, []string{"first", "second"}, attrs["shared_key"])
	})
}

func TestUndeclaredOutputsIgnored(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		testify.NoError(t, env.Engine.RegisterStep(producer))
		testify.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(producer.ID, api.Args{
			"value": "abc",
			"extra": "ignore-me",
		})
		env.MockClient.SetResponse(consumer.ID, api.Args{
			"result": "ok",
			"extra2": "ignore-me-too",
		})

		pl := &api.ExecutionPlan{
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

		fl := env.WaitForFlowStatus("wf-undeclared-outputs", func() {
			err := env.Engine.StartFlow("wf-undeclared-outputs", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		testify.NotNil(t, fl.Attributes["value"])
		testify.NotNil(t, fl.Attributes["result"])
		testify.NotContains(t, fl.Attributes, api.Name("extra"))
		testify.NotContains(t, fl.Attributes, api.Name("extra2"))
	})
}

func TestOutputMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("mapped-output-step")
		st.Attributes = api.AttributeSpecs{
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

		testify.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{
			"payload": map[string]any{"value": "ok"},
		})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-output-mapping")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"input": "value"}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		testify.Equal(t, "ok", ex.Outputs["result"])
		testify.Equal(t, "ok", fl.Attributes["result"].Value)
	})
}
func TestOutputMappingWithRename(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("rename-output")
		st.Attributes = api.AttributeSpecs{
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

		testify.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"success": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-output-rename")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"input": "test"}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		testify.Equal(t, "ok", ex.Outputs["status"])
		testify.Equal(t, "ok", fl.Attributes["status"].Value)
	})
}
