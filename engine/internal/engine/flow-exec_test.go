package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestLinearFlowCompletes(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()
	env.Engine.Start()

	ctx := context.Background()

	producer := &api.Step{
		ID:      "producer",
		Name:    "Producer",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: api.AttributeSpecs{
			"value": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	consumer := &api.Step{
		ID:      "consumer",
		Name:    "Consumer",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: api.AttributeSpecs{
			"value":  {Role: api.RoleRequired, Type: api.TypeString},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, producer))
	assert.NoError(t, env.Engine.RegisterStep(ctx, consumer))

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
		ctx, "wf-linear", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	flow := env.WaitForFlowStatus(t, ctx, "wf-linear", testTimeout)
	assert.Equal(t, api.FlowCompleted, flow.Status)

	assert.Equal(t, api.StepCompleted, flow.Executions[producer.ID].Status)
	assert.Equal(t, api.StepCompleted, flow.Executions[consumer.ID].Status)
	assert.Equal(t, "ok", flow.Attributes["result"].Value)
}
