package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestLinearFlowCompletes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

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

		env.MockClient.SetResponse(producer.ID, api.Args{"value": "abc"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

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

		err := env.Engine.StartFlow(
			"wf-linear", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-linear", testTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		assert.Equal(t, api.StepCompleted, flow.Executions[producer.ID].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[consumer.ID].Status)
		assert.Equal(t, "ok", flow.Attributes["result"].Value)
	})
}

func TestUndeclaredOutputsIgnored(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

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

		err := env.Engine.StartFlow(
			"wf-undeclared-outputs", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-undeclared-outputs", testTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		assert.NotNil(t, flow.Attributes["value"])
		assert.NotNil(t, flow.Attributes["result"])
		assert.NotContains(t, flow.Attributes, api.Name("extra"))
		assert.NotContains(t, flow.Attributes, api.Name("extra2"))
	})
}

func TestPendingUnusedSkip(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		providerA := &api.Step{
			ID:   "provider-a",
			Name: "Provider A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		providerB := &api.Step{
			ID:   "provider-b",
			Name: "Provider B",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed": {Role: api.RoleRequired, Type: api.TypeString},
				"opt":  {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt":    {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"opt": "value"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "done"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				providerA.ID: providerA,
				providerB.ID: providerB,
				consumer.ID:  consumer,
			},
			Attributes: api.AttributeGraph{
				"opt": {
					Providers: []api.StepID{providerA.ID, providerB.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		err := env.Engine.StartFlow(
			"wf-skip-unneeded", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-skip-unneeded", testTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[providerA.ID].Status)
		assert.Equal(t, api.StepSkipped, flow.Executions[providerB.ID].Status)
		assert.Equal(t,
			"outputs not needed", flow.Executions[providerB.ID].Error,
		)
		assert.Equal(t, api.StepCompleted, flow.Executions[consumer.ID].Status)
	})
}
