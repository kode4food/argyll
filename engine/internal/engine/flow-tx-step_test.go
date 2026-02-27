package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
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

		flow := env.WaitForFlowStatus("wf-linear", func() {
			err := env.Engine.StartFlow("wf-linear", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		assert.Equal(t, api.StepCompleted, flow.Executions[producer.ID].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[consumer.ID].Status)
		assert.Equal(t, "ok", flow.Attributes["result"].Value)
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

		flow := env.WaitForFlowStatus("wf-skip-unneeded", func() {
			err := env.Engine.StartFlow("wf-skip-unneeded", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[providerA.ID].Status)
		assert.Equal(t, api.StepSkipped, flow.Executions[providerB.ID].Status)
		assert.Equal(t,
			"outputs not needed", flow.Executions[providerB.ID].Error,
		)
		assert.Equal(t, api.StepCompleted, flow.Executions[consumer.ID].Status)
	})
}

func TestMemoizableStepUsesCache(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewTestStep()
		step.Memoizable = true
		assert.NoError(t, env.Engine.RegisterStep(step))

		env.MockClient.SetResponse(step.ID, api.Args{"output": "cached"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-memo-1", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-1", plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		flow = env.WaitForFlowStatus("wf-memo-2", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-2", plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
	})
}

func TestTimeoutDefaultsBeforeProviderCompletes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"fallback"`,
					Timeout: 50,
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseProvider := make(chan struct{})
		env.MockClient.SetHandler(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{"opt": "real"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				provider.ID: provider,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"opt": {
					Providers: []api.StepID{provider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"seed": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-default")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"seed": "x"}),
		))

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)
		assert.True(t,
			env.MockClient.WaitForInvocation(consumer.ID, 500*time.Millisecond),
		)

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", flow.Executions[consumer.ID].Inputs["opt"])

		close(releaseProvider)
		flow = env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
	})
}

func TestTimeoutZeroFallsBackImmediately(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"fallback"`,
					Timeout: 0,
				},
				"opt2": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Timeout: 0,
				},
				"opt3": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"fallback3"`,
					Timeout: 60000,
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseProvider := make(chan struct{})
		env.MockClient.SetHandler(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{
					"opt":  "real",
					"opt2": "real2",
				}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				provider.ID: provider,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"opt": {
					Providers: []api.StepID{provider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"opt2": {
					Providers: []api.StepID{provider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"seed": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{consumer.ID},
				},
				"opt3": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-zero")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"seed": "x"}),
		))

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)
		assert.True(t,
			env.MockClient.WaitForInvocation(consumer.ID, 120*time.Millisecond),
		)

		close(releaseProvider)
		flow := env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, "fallback", flow.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t,
			"fallback3", flow.Executions[consumer.ID].Inputs["opt3"],
		)
		_, ok := flow.Executions[consumer.ID].Inputs["opt2"]
		assert.False(t, ok)
	})
}

func TestTimeoutDefaultIsStepLocal(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		fast := &api.Step{
			ID:   "fast",
			Name: "Fast",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"fallback"`,
					Timeout: 50,
				},
				"fast_done": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		strict := &api.Step{
			ID:   "strict",
			Name: "Strict",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt":         {Role: api.RoleRequired, Type: api.TypeString},
				"strict_done": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(fast))
		assert.NoError(t, env.Engine.RegisterStep(strict))

		releaseProvider := make(chan struct{})
		env.MockClient.SetHandler(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{"opt": "real"}, nil
			},
		)
		env.MockClient.SetResponse(fast.ID, api.Args{"fast_done": "ok"})
		env.MockClient.SetResponse(strict.ID, api.Args{"strict_done": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{fast.ID, strict.ID},
			Steps: api.Steps{
				provider.ID: provider,
				fast.ID:     fast,
				strict.ID:   strict,
			},
			Attributes: api.AttributeGraph{
				"opt": {
					Providers: []api.StepID{provider.ID},
					Consumers: []api.StepID{fast.ID, strict.ID},
				},
				"seed": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{fast.ID},
				},
				"fast_done": {
					Providers: []api.StepID{fast.ID},
					Consumers: []api.StepID{},
				},
				"strict_done": {
					Providers: []api.StepID{strict.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-local")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"seed": "x"}),
		))

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)
		assert.True(t,
			env.MockClient.WaitForInvocation(fast.ID, 500*time.Millisecond),
		)
		assert.False(t,
			env.MockClient.WaitForInvocation(strict.ID, 120*time.Millisecond),
		)

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", flow.Executions[fast.ID].Inputs["opt"])
		if _, ok := flow.Attributes["opt"]; ok {
			t.Fatalf("timed optional default leaked into flow attributes")
		}

		close(releaseProvider)
		flow = env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, "real", flow.Executions[strict.ID].Inputs["opt"])
	})
}

func TestTimeoutRequiredsGateFallback(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		userProvider := &api.Step{
			ID:   "user-provider",
			Name: "User Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"user_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		productProvider := &api.Step{
			ID:   "product-provider",
			Name: "Product Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		orderCreator := &api.Step{
			ID:   "order-creator",
			Name: "Order Creator",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"guest"`,
					Timeout: 50,
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(userProvider))
		assert.NoError(t, env.Engine.RegisterStep(productProvider))
		assert.NoError(t, env.Engine.RegisterStep(orderCreator))

		releaseUser := make(chan struct{})
		releaseProduct := make(chan struct{})
		userDone := make(chan struct{})

		env.MockClient.SetHandler(userProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseUser
				close(userDone)
				return api.Args{"user_info": "real-user"}, nil
			},
		)
		env.MockClient.SetHandler(productProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProduct
				return api.Args{"product_info": "real-product"}, nil
			},
		)
		env.MockClient.SetResponse(orderCreator.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{orderCreator.ID},
			Steps: api.Steps{
				userProvider.ID:    userProvider,
				productProvider.ID: productProvider,
				orderCreator.ID:    orderCreator,
			},
			Attributes: api.AttributeGraph{
				"user_info": {
					Providers: []api.StepID{userProvider.ID},
					Consumers: []api.StepID{orderCreator.ID},
				},
				"product_info": {
					Providers: []api.StepID{productProvider.ID},
					Consumers: []api.StepID{orderCreator.ID},
				},
				"result": {
					Providers: []api.StepID{orderCreator.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-waits-required")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan))

		assert.True(t, env.MockClient.WaitForInvocation(
			userProvider.ID, 500*time.Millisecond,
		))
		assert.True(t, env.MockClient.WaitForInvocation(
			productProvider.ID, 500*time.Millisecond,
		))

		assert.False(t, env.MockClient.WaitForInvocation(
			orderCreator.ID, 80*time.Millisecond,
		))
		close(releaseUser)

		select {
		case <-userDone:
		case <-time.After(500 * time.Millisecond):
			assert.Fail(t, "timed out waiting for user provider completion")
		}

		assert.False(t, env.MockClient.WaitForInvocation(
			orderCreator.ID, 100*time.Millisecond,
		))

		close(releaseProduct)
		flow := env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t,
			"real-user", flow.Executions[orderCreator.ID].Inputs["user_info"],
		)
		assert.Equal(t, "real-product",
			flow.Executions[orderCreator.ID].Inputs["product_info"],
		)
	})
}

func TestTimeoutStepReadyAnchor(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		gate := &api.Step{
			ID:   "gate",
			Name: "Gate",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"kickoff": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		userProvider := &api.Step{
			ID:   "user-provider",
			Name: "User Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"kickoff":   {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		orderCreator := &api.Step{
			ID:   "order-creator",
			Name: "Order Creator",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"guest"`,
					Timeout: 50,
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(gate))
		assert.NoError(t, env.Engine.RegisterStep(userProvider))
		assert.NoError(t, env.Engine.RegisterStep(orderCreator))

		releaseGate := make(chan struct{})
		env.MockClient.SetHandler(gate.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseGate
				return api.Args{"kickoff": "go"}, nil
			},
		)
		env.MockClient.SetResponse(
			userProvider.ID, api.Args{"user_info": "real"},
		)
		env.MockClient.SetResponse(orderCreator.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{orderCreator.ID},
			Steps: api.Steps{
				gate.ID:         gate,
				userProvider.ID: userProvider,
				orderCreator.ID: orderCreator,
			},
			Attributes: api.AttributeGraph{
				"kickoff": {
					Providers: []api.StepID{gate.ID},
					Consumers: []api.StepID{userProvider.ID},
				},
				"user_info": {
					Providers: []api.StepID{userProvider.ID},
					Consumers: []api.StepID{orderCreator.ID},
				},
				"product_info": {
					Providers: []api.StepID{},
					Consumers: []api.StepID{orderCreator.ID},
				},
				"result": {
					Providers: []api.StepID{orderCreator.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-step-ready")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"product_info": "real-product"}),
		))

		assert.True(t, env.MockClient.WaitForInvocation(
			gate.ID, 500*time.Millisecond,
		))
		assert.False(t, env.MockClient.WaitForInvocation(
			userProvider.ID, 120*time.Millisecond,
		))
		assert.True(t, env.MockClient.WaitForInvocation(
			orderCreator.ID, 500*time.Millisecond,
		))

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t,
			"guest", flow.Executions[orderCreator.ID].Inputs["user_info"],
		)

		close(releaseGate)
		flow = env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
	})
}

func TestTimeoutFiresAfterRequiredsBecomeReady(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		reqProvider := &api.Step{
			ID:   "req-provider",
			Name: "Req Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"req": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		optProvider := &api.Step{
			ID:   "opt-provider",
			Name: "Opt Provider",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"req": {Role: api.RoleRequired, Type: api.TypeString},
				"opt": {
					Role:    api.RoleOptional,
					Type:    api.TypeString,
					Default: `"fallback"`,
					Timeout: 50,
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}

		assert.NoError(t, env.Engine.RegisterStep(reqProvider))
		assert.NoError(t, env.Engine.RegisterStep(optProvider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseReq := make(chan struct{})
		releaseOpt := make(chan struct{})
		env.MockClient.SetHandler(reqProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseReq
				return api.Args{"req": "real-req"}, nil
			},
		)
		env.MockClient.SetHandler(optProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseOpt
				return api.Args{"opt": "real-opt"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				reqProvider.ID: reqProvider,
				optProvider.ID: optProvider,
				consumer.ID:    consumer,
			},
			Attributes: api.AttributeGraph{
				"req": {
					Providers: []api.StepID{reqProvider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"opt": {
					Providers: []api.StepID{optProvider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("wf-opt-timeout-after-requireds")
		assert.NoError(t, env.Engine.StartFlow(flowID, plan))

		assert.True(t, env.MockClient.WaitForInvocation(
			reqProvider.ID, 500*time.Millisecond,
		))
		assert.True(t, env.MockClient.WaitForInvocation(
			optProvider.ID, 500*time.Millisecond,
		))

		close(releaseReq)
		assert.True(t, env.MockClient.WaitForInvocation(
			consumer.ID, 500*time.Millisecond,
		))

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", flow.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t, "real-req", flow.Executions[consumer.ID].Inputs["req"])

		close(releaseOpt)
		flow = env.WaitForFlowStatus(flowID, func() {})
		assert.Equal(t, api.FlowCompleted, flow.Status)
	})
}
