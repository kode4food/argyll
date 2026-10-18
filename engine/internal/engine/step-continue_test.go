package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
)

const (
	optionalShortWindowMS = 50
	optionalExpiredWait   = 100 * time.Millisecond
	optionalResetWindowMS = 400
	optionalArrivalWait   = 250 * time.Millisecond
	optionalLongWindowMS  = 5000
)

func TestOverdueOptionalRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		fixture := collectTimeoutPlan(api.InputCollectSome)
		delete(fixture.consumer.Attributes, "seed")
		delete(fixture.plan.Attributes, "seed")
		for _, st := range fixture.plan.Steps {
			assert.NoError(t, env.Engine.RegisterStep(st))
		}
		env.MockClient.SetResponse(
			fixture.consumer.ID, api.Args{"result": "ok"},
		)
		fs := api.FlowStep{
			FlowID: "overdue-optional",
			StepID: fixture.providerA.ID,
		}
		assert.NoError(t, env.SeedStartedWork(fs, fixture.plan, "token"))
		assert.NoError(t, env.Engine.RecoverFlow(fs.FlowID))
		time.Sleep(100 * time.Millisecond)
		assert.NoError(t, env.Engine.RecoverFlow(fs.FlowID))
		assert.NoError(t, env.Engine.Start())
		assert.True(t,
			env.MockClient.WaitForInvocation(
				fixture.consumer.ID, time.Second,
			))
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t,
			"fallback", fl.Executions[fixture.consumer.ID].Inputs["opt"])
	})
}

func TestDefaultTimeoutBeforeProvider(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: 50,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseProvider := make(chan struct{})
		env.MockClient.SetInvoke(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{"opt": "real"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
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

		id := api.FlowID("wf-opt-timeout-default")
		waitForWorkStarted(env, id, []api.StepID{
			provider.ID,
			consumer.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			))
		})

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])

		close(releaseProvider)
		fl = env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestTimeoutZeroFallsBackImmediately(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: 0,
					},
				},
				"opt2": {
					Role:     api.RoleOptional,
					Type:     api.TypeString,
					Optional: &api.OptionalConfig{Deadline: 0},
				},
				"opt3": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback3"`,
						Deadline: 60000,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseProvider := make(chan struct{})
		env.MockClient.SetInvoke(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{
					"opt":  "real",
					"opt2": "real2",
				}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
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

		id := api.FlowID("wf-opt-timeout-zero")
		waitForWorkStarted(env, id, []api.StepID{
			provider.ID,
			consumer.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			))
		})

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)
		assert.True(t,
			env.MockClient.WaitForInvocation(consumer.ID, 120*time.Millisecond),
		)

		close(releaseProvider)
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t,
			"fallback3", fl.Executions[consumer.ID].Inputs["opt3"],
		)
		_, ok := fl.Executions[consumer.ID].Inputs["opt2"]
		assert.False(t, ok)
	})
}

func TestTimeoutDefaultIsStepLocal(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := &api.Step{
			ID:   "provider",
			Name: "Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		fast := &api.Step{
			ID:   "fast",
			Name: "Fast",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"seed": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"opt": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: 50,
					},
				},
				"fast_done": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		strict := &api.Step{
			ID:   "strict",
			Name: "Strict",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt":         {Role: api.RoleRequired, Type: api.TypeString},
				"strict_done": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(fast))
		assert.NoError(t, env.Engine.RegisterStep(strict))

		releaseProvider := make(chan struct{})
		env.MockClient.SetInvoke(provider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProvider
				return api.Args{"opt": "real"}, nil
			},
		)
		env.MockClient.SetResponse(fast.ID, api.Args{"fast_done": "ok"})
		env.MockClient.SetResponse(strict.ID, api.Args{"strict_done": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{fast.ID, strict.ID}},
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

		id := api.FlowID("wf-opt-timeout-local")
		waitForWorkStarted(env, id, []api.StepID{
			provider.ID,
			fast.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			))
		})

		assert.True(t,
			env.MockClient.WaitForInvocation(provider.ID, 500*time.Millisecond),
		)
		assert.False(t,
			env.MockClient.WaitForInvocation(strict.ID, 120*time.Millisecond),
		)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.StepPending, fl.Executions[strict.ID].Status)
		assert.Equal(t, "fallback", fl.Executions[fast.ID].Inputs["opt"])
		if _, ok := fl.Attributes["opt"]; ok {
			t.Fatalf("timed optional default leaked into flow attributes")
		}

		close(releaseProvider)
		fl = env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, "real", fl.Executions[strict.ID].Inputs["opt"])
	})
}

func TestTimeoutRequiredsGateFallback(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		userProvider := &api.Step{
			ID:   "user-provider",
			Name: "User Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"user_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		productProvider := &api.Step{
			ID:   "product-provider",
			Name: "Product Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		orderCreator := &api.Step{
			ID:   "order-creator",
			Name: "Order Creator",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"guest"`,
						Deadline: 50,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(userProvider))
		assert.NoError(t, env.Engine.RegisterStep(productProvider))
		assert.NoError(t, env.Engine.RegisterStep(orderCreator))

		releaseUser := make(chan struct{})
		releaseProduct := make(chan struct{})
		userDone := make(chan struct{})

		env.MockClient.SetInvoke(userProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseUser
				close(userDone)
				return api.Args{"user_info": "real-user"}, nil
			},
		)
		env.MockClient.SetInvoke(productProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseProduct
				return api.Args{"product_info": "real-product"}, nil
			},
		)
		env.MockClient.SetResponse(orderCreator.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{orderCreator.ID}},
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

		id := api.FlowID("wf-opt-timeout-waits-required")
		waitForWorkStarted(env, id, []api.StepID{
			userProvider.ID,
			productProvider.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
		})

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
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t,
			"real-user", fl.Executions[orderCreator.ID].Inputs["user_info"],
		)
		assert.Equal(t, "real-product",
			fl.Executions[orderCreator.ID].Inputs["product_info"],
		)
	})
}

func TestTimeoutWaitsForLaterOptional(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA := &api.Step{
			ID:   "provider-a",
			Name: "Provider A",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt_a": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		providerB := &api.Step{
			ID:   "provider-b",
			Name: "Provider B",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt_b": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"seed": {Role: api.RoleRequired, Type: api.TypeString},
				"opt_a": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback-a"`,
						Deadline: 50,
					},
				},
				"opt_b": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback-b"`,
						Deadline: 120,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseA := make(chan struct{})
		releaseB := make(chan struct{})
		env.MockClient.SetInvoke(providerA.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseA
				return api.Args{"opt_a": "real-a"}, nil
			},
		)
		env.MockClient.SetInvoke(providerB.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseB
				return api.Args{"opt_b": "real-b"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
			Steps: api.Steps{
				providerA.ID: providerA,
				providerB.ID: providerB,
				consumer.ID:  consumer,
			},
			Attributes: api.AttributeGraph{
				"opt_a": {
					Providers: []api.StepID{providerA.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"opt_b": {
					Providers: []api.StepID{providerB.ID},
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

		id := api.FlowID("wf-opt-timeout-staggered")
		waitForWorkStarted(env, id, []api.StepID{
			providerA.ID,
			providerB.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			))
		})

		assert.False(t, env.MockClient.WaitForInvocation(
			consumer.ID, 80*time.Millisecond,
		))
		assert.True(t, env.MockClient.WaitForInvocation(
			consumer.ID, 180*time.Millisecond,
		))

		close(releaseA)
		close(releaseB)
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t,
			"fallback-a", fl.Executions[consumer.ID].Inputs["opt_a"],
		)
		assert.Equal(t,
			"fallback-b", fl.Executions[consumer.ID].Inputs["opt_b"],
		)
	})
}

func TestTimeoutStepReadyAnchor(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		gate := &api.Step{
			ID:   "gate",
			Name: "Gate",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"kickoff": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		userProvider := &api.Step{
			ID:   "user-provider",
			Name: "User Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"kickoff":   {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		orderCreator := &api.Step{
			ID:   "order-creator",
			Name: "Order Creator",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"product_info": {Role: api.RoleRequired, Type: api.TypeString},
				"user_info": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"guest"`,
						Deadline: 50,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(gate))
		assert.NoError(t, env.Engine.RegisterStep(userProvider))
		assert.NoError(t, env.Engine.RegisterStep(orderCreator))

		releaseGate := make(chan struct{})
		env.MockClient.SetInvoke(gate.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseGate
				return api.Args{"kickoff": "go"}, nil
			},
		)
		env.MockClient.SetResponse(
			userProvider.ID, api.Args{"user_info": "real"},
		)
		env.MockClient.SetResponse(orderCreator.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{orderCreator.ID}},
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

		id := api.FlowID("wf-opt-timeout-step-ready")
		waitForWorkStarted(env, id, []api.StepID{
			gate.ID,
			orderCreator.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"product_info": {"real-product"}}),
			))
		})

		assert.True(t, env.MockClient.WaitForInvocation(
			gate.ID, 500*time.Millisecond,
		))
		assert.False(t, env.MockClient.WaitForInvocation(
			userProvider.ID, 120*time.Millisecond,
		))
		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			"guest", fl.Executions[orderCreator.ID].Inputs["user_info"],
		)

		close(releaseGate)
		fl = env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestTimeoutAfterRequireds(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		reqProvider := &api.Step{
			ID:   "req-provider",
			Name: "Req Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"req": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		optProvider := &api.Step{
			ID:   "opt-provider",
			Name: "Opt Provider",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"opt": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"req": {Role: api.RoleRequired, Type: api.TypeString},
				"opt": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: 50,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(reqProvider))
		assert.NoError(t, env.Engine.RegisterStep(optProvider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseReq := make(chan struct{})
		releaseOpt := make(chan struct{})
		env.MockClient.SetInvoke(reqProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseReq
				return api.Args{"req": "real-req"}, nil
			},
		)
		env.MockClient.SetInvoke(optProvider.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseOpt
				return api.Args{"opt": "real-opt"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
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

		id := api.FlowID("wf-opt-timeout-after-requireds")
		waitForWorkStarted(env, id, []api.StepID{
			reqProvider.ID,
			optProvider.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
		})

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

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t, "real-req", fl.Executions[consumer.ID].Inputs["req"])

		close(releaseOpt)
		fl = env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestFinalityDeadline(t *testing.T) {
	for _, collect := range []api.InputCollect{
		api.InputCollectLast, api.InputCollectSome,
	} {
		t.Run(string(collect), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				early := helpers.NewStepWithOutputs("early", "req")
				late := helpers.NewStepWithOutputs("late", "req")
				optional := helpers.NewStepWithOutputs("optional", "opt")
				consumer := helpers.NewStepWithOutputs("consumer", "result")
				consumer.Attributes["req"] = &api.AttributeSpec{
					Role:     api.RoleRequired,
					Type:     api.TypeString,
					Required: &api.RequiredConfig{Collect: collect},
				}
				consumer.Attributes["opt"] = &api.AttributeSpec{
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: optionalShortWindowMS,
					},
				}
				for _, st := range []*api.Step{early, late, optional} {
					st.HTTP.Invoke.Mode = api.ActionModeAsync
					env.MockClient.SetResponse(st.ID, api.Args{})
				}
				env.MockClient.SetResponse(
					consumer.ID, api.Args{"result": "ok"},
				)
				assert.NoError(t,
					env.Engine.RegisterSteps(early, late, optional, consumer))
				fid := api.FlowID("provider-finality")
				assert.NoError(t,
					env.Engine.StartFlow(api.CreateFlowRequest{
						ID:    fid,
						Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
					}))
				complete := func(sid api.StepID, outputs api.Args) {
					fl, err := env.Engine.GetFlowState(fid)
					assert.NoError(t, err)
					items := fl.Executions[sid].WorkItems
					assert.Len(t, items, 1)
					for token := range items {
						assert.NoError(t,
							env.Engine.CompleteWork(
								api.FlowStep{FlowID: fid, StepID: sid},
								token, outputs,
							))
					}
				}
				complete(early.ID, api.Args{"req": "x"})
				time.Sleep(optionalExpiredWait)
				complete(optional.ID, api.Args{"opt": "real"})
				fl, err := env.Engine.GetFlowState(fid)
				assert.NoError(t, err)
				assert.Equal(t,
					api.StepPending, fl.Executions[consumer.ID].Status)
				complete(late.ID, api.Args{})
				fl = env.WaitForTerminalFlow(fid)
				assert.Equal(t, api.FlowCompleted, fl.Status)
				assert.Equal(t,
					"real", fl.Executions[consumer.ID].Inputs["opt"])
			})
		})
	}
}

func TestOptionalWindowResets(t *testing.T) {
	for _, collect := range []api.InputCollect{
		api.InputCollectLast, api.InputCollectSome, api.InputCollectAll,
	} {
		t.Run(string(collect), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				pl := optionalWindowPlan(collect, optionalResetWindowMS)
				startOptionalWindow(t, env, pl)
				time.Sleep(optionalArrivalWait)
				completeOptionalWork(t, env, "a", api.Args{"opt": "a"})
				assert.NoError(t, env.Engine.RecoverFlow("window"))
				time.Sleep(optionalArrivalWait)
				completeOptionalWork(t, env, "b", api.Args{"opt": "b"})
				fl, err := env.Engine.GetFlowState("window")
				assert.NoError(t, err)
				assert.Equal(t,
					api.StepPending, fl.Executions["consumer"].Status)
				completeOptionalWork(t, env, "c", api.Args{"opt": "c"})
				fl = env.WaitForTerminalFlow("window")
				assert.Equal(t, api.FlowCompleted, fl.Status)
				if collect == api.InputCollectLast {
					assert.Equal(t,
						"c", fl.Executions["consumer"].Inputs["opt"])
					return
				}
				assert.Equal(t,
					[]any{"a", "b", "c"},
					fl.Executions["consumer"].Inputs["opt"])
			})
		})
	}
}

func TestClosedOptionalWindow(t *testing.T) {
	for _, collect := range []api.InputCollect{
		api.InputCollectLast, api.InputCollectSome, api.InputCollectAll,
		api.InputCollectFirst,
	} {
		t.Run(string(collect), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				pl := optionalWindowPlan(collect, optionalShortWindowMS)
				addOptionalHold(pl)
				startOptionalWindow(t, env, pl)
				if collect != api.InputCollectFirst {
					completeOptionalWork(t, env, "a", api.Args{"opt": "a"})
				}
				time.Sleep(optionalExpiredWait)
				if collect == api.InputCollectFirst {
					completeOptionalWork(t, env, "a", api.Args{"opt": "late"})
				}
				completeOptionalWork(t, env, "b", api.Args{"opt": "late"})
				completeOptionalWork(t, env, "c", api.Args{"opt": "late"})
				assert.NoError(t, env.Engine.RecoverFlow("window"))
				fl, err := env.Engine.GetFlowState("window")
				assert.NoError(t, err)
				assert.Equal(t,
					api.StepPending, fl.Executions["consumer"].Status)
				completeOptionalWork(t, env, "hold", api.Args{"hold": "ready"})
				fl = env.WaitForTerminalFlow("window")
				assert.Equal(t,
					"fallback", fl.Executions["consumer"].Inputs["opt"])
				assert.Equal(t,
					"ready", fl.Executions["consumer"].Inputs["hold"])
			})
		})
	}
}

func TestOptionalZeroSnapshot(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		pl := optionalWindowPlan(api.InputCollectFirst, 0)
		addOptionalHold(pl)
		startOptionalWindow(t, env, pl)
		completeOptionalWork(t, env, "a", api.Args{"opt": "late"})
		completeOptionalWork(t, env, "b", api.Args{"opt": "late"})
		completeOptionalWork(t, env, "c", api.Args{"opt": "late"})
		completeOptionalWork(t, env, "hold", api.Args{"hold": "ready"})
		fl := env.WaitForTerminalFlow("window")
		assert.Equal(t, "fallback", fl.Executions["consumer"].Inputs["opt"])
	})
}

func TestImpossibleOptional(t *testing.T) {
	for _, collect := range []api.InputCollect{
		api.InputCollectAll, api.InputCollectNone,
	} {
		t.Run(string(collect), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				pl := optionalWindowPlan(collect, optionalLongWindowMS)
				startOptionalWindow(t, env, pl)
				out := api.Args{}
				if collect == api.InputCollectNone {
					out["opt"] = "present"
				}
				env.WaitFor(
					wait.StepTerminal(api.FlowStep{
						FlowID: "window",
						StepID: "consumer",
					}),
					func() { completeOptionalWork(t, env, "a", out) },
				)
				fl, err := env.Engine.GetFlowState("window")
				assert.NoError(t, err)
				assert.Equal(t,
					"fallback", fl.Executions["consumer"].Inputs["opt"])
				completeOptionalWork(t, env, "b", api.Args{})
				completeOptionalWork(t, env, "c", api.Args{})
				fl = env.WaitForTerminalFlow("window")
				assert.Equal(t, api.FlowCompleted, fl.Status)
			})
		})
	}
}

func TestOptionalNoneInit(t *testing.T) {
	for _, present := range []bool{false, true} {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			st := helpers.NewStepWithOutputs("consumer", "result")
			st.Attributes["opt"] = &api.AttributeSpec{
				Role: api.RoleOptional,
				Type: api.TypeString,
				Optional: &api.OptionalConfig{
					Collect:  api.InputCollectNone,
					Deadline: optionalLongWindowMS,
					Default:  `"fallback"`,
				},
			}
			assert.NoError(t, env.Engine.RegisterStep(st))
			env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})
			init := api.InitArgs{}
			if present {
				init["opt"] = []any{"blocked"}
			}
			assert.NoError(t,
				env.Engine.StartFlow(api.CreateFlowRequest{
					ID:    "none-init",
					Goals: api.Goals{Steps: []api.StepID{st.ID}},
					Init:  init,
				}))
			fl := env.WaitForTerminalFlow("none-init")
			assert.Equal(t, api.FlowCompleted, fl.Status)
			if present {
				assert.Equal(t, "fallback", fl.Executions[st.ID].Inputs["opt"])
				return
			}
			assert.NotContains(t, fl.Executions[st.ID].Inputs, api.Name("opt"))
		})
	}
}

func TestOptionalLastTimeout(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		fixture := collectTimeoutPlan(api.InputCollectLast)
		providerA := fixture.providerA
		providerB := fixture.providerB
		consumer := fixture.consumer
		pl := fixture.plan

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"opt": "real"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		id := api.FlowID("wf-opt-last-timeout")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t, api.StepSkipped, fl.Executions[providerB.ID].Status)
		assert.Equal(t,
			"outputs not needed", fl.Executions[providerB.ID].Error,
		)
	})
}

func TestOptionalSomeTimeout(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		fixture := collectTimeoutPlan(api.InputCollectSome)
		providerA := fixture.providerA
		providerB := fixture.providerB
		consumer := fixture.consumer
		pl := fixture.plan

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"opt": "real"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		id := api.FlowID("wf-opt-some-timeout")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"seed": {"x"}}),
			)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])
		assert.Equal(t, api.StepSkipped, fl.Executions[providerB.ID].Status)
	})
}

func TestTimeoutAfterProviderFails(t *testing.T) {
	if testing.Short() {
		t.Skip("waits on a real optional deadline")
	}
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		failing := helpers.NewStepWithOutputs("failing-provider", "gate")
		slow := helpers.NewStepWithOutputs("slow-provider", "opt")
		consumer := &api.Step{
			ID:   "consumer",
			Name: "Consumer",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"gate": {
					Role: api.RoleRequired,
					Type: api.TypeString,
					Required: &api.RequiredConfig{
						Collect: api.InputCollectNone,
					},
				},
				"opt": {
					Role: api.RoleOptional,
					Type: api.TypeString,
					Optional: &api.OptionalConfig{
						Default:  `"fallback"`,
						Deadline: 200,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
		}
		for _, st := range []*api.Step{failing, slow, consumer} {
			assert.NoError(t, env.Engine.RegisterStep(st))
		}

		release := make(chan struct{})
		defer close(release)
		env.MockClient.SetError(failing.ID, assert.AnError)
		env.MockClient.SetInvoke(slow.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-release
				return api.Args{"opt": "late"}, nil
			},
		)
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
			Steps: api.Steps{
				failing.ID:  failing,
				slow.ID:     slow,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"gate": {
					Providers: []api.StepID{failing.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"opt": {
					Providers: []api.StepID{slow.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("wf-timeout-provider-fails")
		fs := api.FlowStep{FlowID: id, StepID: consumer.ID}
		env.WaitFor(wait.StepTerminal(fs), func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.StepCompleted, fl.Executions[consumer.ID].Status)
		assert.Equal(t, "fallback", fl.Executions[consumer.ID].Inputs["opt"])
	})
}

func waitForWorkStarted(
	env *helpers.TestEngineEnv, fid api.FlowID, steps []api.StepID,
	fn func(),
) {
	filters := make([]wait.EventFilter, len(steps))
	for idx, sid := range steps {
		filters[idx] = wait.WorkStarted(api.FlowStep{
			FlowID: fid,
			StepID: sid,
		})
	}
	env.WithConsumer(func(consumer *event.Consumer) {
		w := wait.On(env.T, consumer)
		fn()
		w.ForAll(filters...)
	})
}

type collectTimeoutPlanRes struct {
	providerA *api.Step
	providerB *api.Step
	consumer  *api.Step
	plan      *api.ExecutionPlan
}

func collectTimeoutPlan(collect api.InputCollect) collectTimeoutPlanRes {
	providerA := &api.Step{
		ID:   "provider-a",
		Name: "Provider A",
		Type: api.StepTypeService,
		Attributes: api.AttributeSpecs{
			"opt": {Role: api.RoleOutput, Type: api.TypeAny},
		},
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.com"},
		},
	}
	providerB := &api.Step{
		ID:   "provider-b",
		Name: "Provider B",
		Type: api.StepTypeService,
		Attributes: api.AttributeSpecs{
			"missing": {Role: api.RoleRequired, Type: api.TypeString},
			"opt":     {Role: api.RoleOutput, Type: api.TypeAny},
		},
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.com"},
		},
	}
	consumer := &api.Step{
		ID:   "consumer",
		Name: "Consumer",
		Type: api.StepTypeService,
		Attributes: api.AttributeSpecs{
			"seed": {Role: api.RoleRequired, Type: api.TypeString},
			"opt": {
				Role: api.RoleOptional,
				Type: api.TypeAny,
				Optional: &api.OptionalConfig{
					Collect:  collect,
					Default:  `"fallback"`,
					Deadline: 50,
				},
			},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.com"},
		},
	}
	pl := &api.ExecutionPlan{
		Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
		Steps: api.Steps{
			providerA.ID: providerA,
			providerB.ID: providerB,
			consumer.ID:  consumer,
		},
		Attributes: api.AttributeGraph{
			"missing": {
				Providers: []api.StepID{},
				Consumers: []api.StepID{providerB.ID},
			},
			"opt": {
				Providers: []api.StepID{providerA.ID, providerB.ID},
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
	return collectTimeoutPlanRes{
		providerA: providerA,
		providerB: providerB,
		consumer:  consumer,
		plan:      pl,
	}
}

func optionalWindowPlan(
	collect api.InputCollect, timeout int64,
) *api.ExecutionPlan {
	consumer := helpers.NewStepWithOutputs("consumer", "result")
	consumer.Attributes["seed"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}
	consumer.Attributes["opt"] = &api.AttributeSpec{
		Role: api.RoleOptional,
		Type: api.TypeString,
		Optional: &api.OptionalConfig{
			Collect:  collect,
			Deadline: timeout,
			Default:  `"fallback"`,
		},
	}
	pl := &api.ExecutionPlan{
		Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
		Steps: api.Steps{consumer.ID: consumer},
		Attributes: api.AttributeGraph{
			"opt": {
				Providers: []api.StepID{"a", "b", "c"},
				Consumers: []api.StepID{consumer.ID},
			},
			"seed": {Consumers: []api.StepID{consumer.ID}},
		},
	}
	for _, sid := range []api.StepID{"a", "b", "c"} {
		pl.Steps[sid] = helpers.NewStepWithOutputs(sid, "opt")
	}
	return pl
}

func addOptionalHold(pl *api.ExecutionPlan) {
	pl.Steps["consumer"].Attributes["hold"] = &api.AttributeSpec{
		Role: api.RoleOptional,
		Type: api.TypeString,
		Optional: &api.OptionalConfig{
			Deadline: optionalLongWindowMS,
			Default:  `"missing"`,
		},
	}
	pl.Steps["hold"] = helpers.NewStepWithOutputs("hold", "hold")
	pl.Attributes["hold"] = &api.AttributeEdges{
		Providers: []api.StepID{"hold"},
		Consumers: []api.StepID{"consumer"},
	}
}

func startOptionalWindow(
	t *testing.T, env *helpers.TestEngineEnv, pl *api.ExecutionPlan,
) {
	t.Helper()
	assert.NoError(t, env.Engine.Start())
	var providers []api.StepID
	for sid, st := range pl.Steps {
		if sid != "consumer" {
			st.HTTP.Invoke.Mode = api.ActionModeAsync
			env.MockClient.SetResponse(sid, api.Args{})
			providers = append(providers, sid)
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
	}
	env.MockClient.SetResponse("consumer", api.Args{"result": "ok"})
	waitForWorkStarted(env, "window", providers, func() {
		assert.NoError(t,
			env.Engine.StartPlan(
				"window", pl, flow.WithInit(api.InitArgs{"seed": {"ready"}}),
			))
	})
}

func completeOptionalWork(
	t *testing.T, env *helpers.TestEngineEnv, sid api.StepID, outputs api.Args,
) {
	t.Helper()
	fl, err := env.Engine.GetFlowState("window")
	assert.NoError(t, err)
	items := fl.Executions[sid].WorkItems
	assert.Len(t, items, 1)
	for token := range items {
		assert.NoError(t,
			env.Engine.CompleteWork(
				api.FlowStep{FlowID: fl.ID, StepID: sid}, token, outputs,
			))
	}
}
