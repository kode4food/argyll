package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestLinearFlowCompletes(t *testing.T) {
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

		env.MockClient.SetResponse(producer.ID, api.Args{"value": "abc"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

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

		fl := env.WaitForFlowStatus("wf-linear", func() {
			err := env.Engine.StartFlow("wf-linear", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		assert.Equal(t, api.StepCompleted, fl.Executions[producer.ID].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[consumer.ID].Status)
		assert.Equal(t, "ok", fl.Attributes["result"][0].Value)
	})
}
func TestPendingUnusedSkip(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

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

		pl := &api.ExecutionPlan{
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

		fl := env.WaitForFlowStatus("wf-skip-unneeded", func() {
			err := env.Engine.StartFlow("wf-skip-unneeded", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[providerA.ID].Status)
		assert.Equal(t, api.StepSkipped, fl.Executions[providerB.ID].Status)
		assert.Equal(t,
			"outputs not needed", fl.Executions[providerB.ID].Error,
		)
		assert.Equal(t, api.StepCompleted, fl.Executions[consumer.ID].Status)
	})
}

func TestSkipFailedAllProvider(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA := &api.Step{
			ID:   "provider-a",
			Name: "Provider A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed": {Role: api.RoleRequired, Type: api.TypeString},
				"data": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		providerB := &api.Step{
			ID:   "provider-b",
			Name: "Provider B",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"data": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"data": {
					Role:  api.RoleRequired,
					Type:  api.TypeString,
					Input: &api.InputConfig{Collect: api.InputCollectAll},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerB.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				providerA.ID: providerA,
				providerB.ID: providerB,
				consumer.ID:  consumer,
			},
			Attributes: api.AttributeGraph{
				"seed": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{providerA.ID},
				},
				"data": {
					Providers: []api.StepID{providerA.ID, providerB.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("wf-skip-all")
		env.WaitFor(wait.StepTerminal(api.FlowStep{
			FlowID: id,
			StepID: providerA.ID,
		}), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.StepSkipped, fl.Executions[providerA.ID].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[providerB.ID].Status)
	})
}

func TestSkipStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create a step with a predicate that returns false, causing a  skip
		st := helpers.NewStepWithPredicate(
			"step-skip", api.ScriptLangAle, "false",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-skip"},
			Steps: api.Steps{st.ID: st},
		}

		// Wait for step to be skipped
		env.WaitFor(wait.StepTerminal(api.FlowStep{
			FlowID: "wf-skip",
			StepID: "step-skip",
		}), func() {
			err = env.Engine.StartFlow("wf-skip", pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState("wf-skip")
		assert.NoError(t, err)
		ex := fl.Executions["step-skip"]
		assert.NotNil(t, ex)
		assert.Equal(t, api.StepSkipped, ex.Status)
		assert.Equal(t, "predicate returned false", ex.Error)
	})
}
