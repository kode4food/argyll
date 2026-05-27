package engine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestOptionalDefaults(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("default-step")
		st.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role:     api.RoleOptional,
				Type:     api.TypeString,
				Optional: &api.OptionalConfig{Default: `"fallback"`},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-defaults")
		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"input": {"value"}}),
			)
			assert.NoError(t, err)
			w.ForAll(
				wait.WorkSucceeded(api.FlowStep{
					FlowID: id,
					StepID: st.ID,
				}),
				wait.FlowTerminal(id),
			)
		})
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, "value", ex.Inputs["input"])
		assert.Equal(t, "fallback", ex.Inputs["optional"])
	})
}

func TestCollectFirst(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("collect-first")
		st.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeAny,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetHandler(st.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, "a", args["input"])
				return api.Args{"result": "ok"}, nil
			},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-collect-first")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"input": {"a", "b"}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestCollectLast(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA, providerB, consumer, pl := collectPlan(
			"last", api.InputCollectLast,
		)
		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		releaseA := make(chan struct{})
		releaseB := make(chan struct{})
		env.MockClient.SetHandler(providerA.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseA
				return api.Args{"data": "a"}, nil
			},
		)
		env.MockClient.SetHandler(providerB.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				<-releaseB
				return api.Args{"data": "b"}, nil
			},
		)
		env.MockClient.SetHandler(consumer.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, "b", args["data"])
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-collect-last")
		waitForWorkStarted(env, id, []api.StepID{
			providerA.ID,
			providerB.ID,
		}, func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})
		assert.True(t,
			env.MockClient.WaitForInvocation(
				providerA.ID, 500*time.Millisecond,
			),
		)
		assert.True(t,
			env.MockClient.WaitForInvocation(
				providerB.ID, 500*time.Millisecond,
			),
		)

		close(releaseA)
		assert.False(t,
			env.MockClient.WaitForInvocation(consumer.ID, 100*time.Millisecond),
		)

		close(releaseB)
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestCollectLists(t *testing.T) {
	tests := []struct {
		name    string
		collect api.InputCollect
	}{
		{name: "some", collect: api.InputCollectSome},
		{name: "all", collect: api.InputCollectAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				assert.NoError(t, env.Engine.Start())

				providerA, providerB, consumer, pl := collectPlan(
					tt.name, tt.collect,
				)
				assert.NoError(t, env.Engine.RegisterStep(providerA))
				assert.NoError(t, env.Engine.RegisterStep(providerB))
				assert.NoError(t, env.Engine.RegisterStep(consumer))

				env.MockClient.SetResponse(providerA.ID, api.Args{"data": "a"})
				env.MockClient.SetResponse(providerB.ID, api.Args{"data": "b"})
				env.MockClient.SetHandler(consumer.ID,
					func(
						_ *api.Step, args api.Args, _ api.Metadata,
					) (api.Args, error) {
						assert.ElementsMatch(t, []any{"a", "b"}, args["data"])
						return api.Args{"result": "ok"}, nil
					},
				)

				id := api.FlowID("wf-collect-" + tt.name)
				fl := env.WaitForFlowStatus(id, func() {
					err := env.Engine.StartFlow(id, pl)
					assert.NoError(t, err)
				})
				assert.Equal(t, api.FlowCompleted, fl.Status)
			})
		})
	}
}

func TestCollectNone(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		provider := collectProvider("provider-none")
		consumer := &api.Step{
			ID:   "consumer-none",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"data": {
					Role: api.RoleOptional,
					Type: api.TypeAny,
					Optional: &api.OptionalConfig{
						Collect: api.InputCollectNone,
						Default: `"fallback"`,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				provider.ID: provider,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"data": {
					Providers: []api.StepID{provider.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(provider))
		assert.NoError(t, env.Engine.RegisterStep(consumer))
		env.MockClient.SetResponse(provider.ID, api.Args{})
		env.MockClient.SetHandler(consumer.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, "fallback", args["data"])
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-collect-none")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestCollectNoneNoProvider(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "consumer-none-no-provider",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"data": {
					Role: api.RoleRequired,
					Type: api.TypeAny,
					Required: &api.RequiredConfig{
						Collect: api.InputCollectNone,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetHandler(st.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.NotContains(t, args, "data")
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-collect-none-no-provider")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestCollectSomeInit(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "consumer-some-init-no-provider",
			Name: "Consumer",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"data": {
					Role: api.RoleRequired,
					Type: api.TypeAny,
					Required: &api.RequiredConfig{
						Collect: api.InputCollectSome,
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
		}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetHandler(st.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, []any{"ready"}, args["data"])
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-collect-some-init-no-provider")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"data": {"ready"}}))
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestConstObject(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("const-object")
		st.Attributes = api.AttributeSpecs{
			"config": {
				Role:  api.RoleConst,
				Type:  api.TypeObject,
				Const: &api.ConstConfig{Value: `{"name":"cfg","count":2}`},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-const-object")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		cfg, ok := ex.Inputs["config"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, map[string]any{
			"name":  "cfg",
			"count": float64(2),
		}, cfg)
	})
}

func TestInputMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("mapped-input-step")
		st.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeObject,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$.foo",
						},
					},
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-input-mapping")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{
					"input": {map[string]any{"foo": "value"}},
				}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, "value", ex.Inputs["input"])
	})
}
func TestInputRename(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("rename-input")
		st.Attributes = api.AttributeSpecs{
			"user_email": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{Name: "email"},
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-input-rename")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"user_email": {"test@example.com"}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}
func TestPredicateFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"pred-fail", api.ScriptLangLua, "error('boom')",
		)

		assert.NoError(t, env.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		ex := env.WaitForStepStatus("wf-pred-fail", st.ID, func() {
			err := env.Engine.StartFlow("wf-pred-fail", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.StepFailed, ex.Status)
		assert.True(t, strings.Contains(ex.Error, "predicate"))
	})
}

func TestJPathNullMatch(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"jpath-null", api.ScriptLangJPath, "$.flag", "result",
		)
		st.Attributes["flag"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeAny,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-jpath-null", func() {
			err := env.Engine.StartFlow("wf-jpath-null", pl,
				flow.WithInit(api.InitArgs{"flag": {nil}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[st.ID].Status)

		assert.True(t, env.MockClient.WasInvoked(st.ID))
	})
}

func TestMatchRoutes(t *testing.T) {
	tests := []struct {
		name           string
		route          string
		init           api.InitArgs
		wantEmail      bool
		wantCustomer   bool
		wantPostal     bool
		wantFlowStatus api.FlowStatus
	}{
		{
			name:  "email",
			route: "email",
			init: api.InitArgs{
				"email_address": {"user@example.com"},
			},
			wantEmail:      true,
			wantFlowStatus: api.FlowCompleted,
		},
		{
			name:  "postal",
			route: "postal",
			init: api.InitArgs{
				"customer_id": {"cust-1"},
			},
			wantCustomer:   true,
			wantPostal:     true,
			wantFlowStatus: api.FlowCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				assert.NoError(t, env.Engine.Start())

				route, customer, email, postal := routingSteps()
				for _, st := range []*api.Step{route, customer, email, postal} {
					assert.NoError(t, env.Engine.RegisterStep(st))
				}
				env.MockClient.SetResponse(route.ID,
					api.Args{"notification_type": tt.route},
				)
				env.MockClient.SetResponse(customer.ID,
					api.Args{"customer": map[string]any{"id": "cust-1"}},
				)
				env.MockClient.SetResponse(email.ID,
					api.Args{"email_result": "sent"},
				)
				env.MockClient.SetResponse(postal.ID,
					api.Args{"postal_result": "sent"},
				)

				cat := api.CatalogState{
					Steps: api.Steps{
						route.ID:    route,
						customer.ID: customer,
						email.ID:    email,
						postal.ID:   postal,
					},
					Attributes: api.AttributeGraph{}.
						AddStep(route).
						AddStep(customer).
						AddStep(email).
						AddStep(postal),
				}
				pl, err := plan.Create(
					env.Engine.Matcher, env.Engine.Children, cat,
					[]api.StepID{email.ID, postal.ID}, tt.init,
				)
				assert.NoError(t, err)

				id := api.FlowID("wf-match-" + tt.name)
				fl := env.WaitForFlowStatus(id, func() {
					err := env.Engine.StartFlow(id, pl,
						flow.WithInit(tt.init),
					)
					assert.NoError(t, err)
				})
				assert.Equal(t, tt.wantFlowStatus, fl.Status)
				assert.Equal(t, tt.wantEmail,
					env.MockClient.WasInvoked(email.ID),
				)
				assert.Equal(t, tt.wantCustomer,
					env.MockClient.WasInvoked(customer.ID),
				)
				assert.Equal(t, tt.wantPostal,
					env.MockClient.WasInvoked(postal.ID),
				)
			})
		})
	}
}

func TestMatchFilters(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA, providerB, consumer, pl := collectPlan(
			"match-some", api.InputCollectSome,
		)
		consumer.Attributes["data"].Required.Match = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return value == "a"`,
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"data": "a"})
		env.MockClient.SetResponse(providerB.ID, api.Args{"data": "b"})
		env.MockClient.SetHandler(consumer.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, []any{"a"}, args["data"])
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-match-some")
		fl := env.WaitForFlowStatus(id, func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestMatchFirstPending(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA, providerB, consumer, pl := collectPlan(
			"match-first", api.InputCollectFirst,
		)
		consumer.Attributes["data"].Required.Match = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return value == "a"`,
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"data": "b"})
		env.MockClient.SetResponse(providerB.ID, api.Args{"data": "a"})
		env.MockClient.SetHandler(consumer.ID,
			func(_ *api.Step, args api.Args, _ api.Metadata) (api.Args, error) {
				assert.Equal(t, "a", args["data"])
				return api.Args{"result": "ok"}, nil
			},
		)

		id := api.FlowID("wf-match-first")
		fl := env.WaitForFlowStatus(id, func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestMatchAllPrunes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		providerA, providerB, consumer, pl := collectPlan(
			"match-all", api.InputCollectAll,
		)
		consumer.Attributes["data"].Required.Match = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return value == "a"`,
		}

		assert.NoError(t, env.Engine.RegisterStep(providerA))
		assert.NoError(t, env.Engine.RegisterStep(providerB))
		assert.NoError(t, env.Engine.RegisterStep(consumer))

		env.MockClient.SetResponse(providerA.ID, api.Args{"data": "a"})
		env.MockClient.SetResponse(providerB.ID, api.Args{"data": "b"})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "ok"})

		id := api.FlowID("wf-match-all")
		err := env.Engine.StartFlow(id, pl)
		assert.NoError(t, err)
		fl := env.WaitForTerminalFlow(id)
		assert.False(t, env.MockClient.WasInvoked(consumer.ID))
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.Equal(t, api.StepSkipped, fl.Executions[consumer.ID].Status)
		assert.Equal(t,
			[]api.Name{"data"}, fl.Executions[consumer.ID].Unsatisfied,
		)
	})
}

func TestMatchSkipInputs(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		consumer := helpers.NewSimpleStep("notification")
		consumer.Attributes = api.AttributeSpecs{
			"user_info": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"payment_result": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Match: &api.ScriptConfig{
						Language: api.ScriptLangLua,
						Script:   `return value == "paid"`,
					},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(consumer))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{consumer.ID: consumer},
		}

		id := api.FlowID("wf-required-match-skip-inputs")
		err := env.Engine.StartFlow(id, pl,
			flow.WithInit(api.InitArgs{
				"user_info":      {"resolved-user"},
				"payment_result": {"declined"},
			}),
		)
		assert.NoError(t, err)

		fl := env.WaitForTerminalFlow(id)
		ex := fl.Executions[consumer.ID]
		assert.Equal(t, api.StepSkipped, ex.Status)
		assert.Equal(t, "resolved-user", ex.Inputs["user_info"])
		_, hasPaymentResult := ex.Inputs["payment_result"]
		assert.False(t, hasPaymentResult)
		assert.Equal(t, []api.Name{"payment_result"}, ex.Unsatisfied)
	})
}

func TestInputAle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("ale-input-map")
		st.Attributes = api.AttributeSpecs{
			"amount": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangAle,
							Script:   "(* amount 2)",
						},
					},
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": float64(10)})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-ale-input")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"amount": {float64(5)}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, float64(10), ex.Inputs["amount"])
	})
}

func routingSteps() (*api.Step, *api.Step, *api.Step, *api.Step) {
	route := &api.Step{
		ID:   "route",
		Name: "Route",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"notification_type": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	customer := &api.Step{
		ID:   "customer-lookup",
		Name: "Customer Lookup",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"customer_id": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"customer": {
				Role: api.RoleOutput,
				Type: api.TypeObject,
			},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	email := &api.Step{
		ID:   "send-email",
		Name: "Send Email",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"notification_type": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Match: &api.ScriptConfig{
						Language: api.ScriptLangLua,
						Script:   `return value == "email"`,
					},
				},
			},
			"email_address": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"email_result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	postal := &api.Step{
		ID:   "send-postal",
		Name: "Send Postal",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"notification_type": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Match: &api.ScriptConfig{
						Language: api.ScriptLangLua,
						Script:   `return value == "postal"`,
					},
				},
			},
			"customer": {
				Role: api.RoleRequired,
				Type: api.TypeObject,
			},
			"postal_result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	return route, customer, email, postal
}
func TestPredicateExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangAle, "true", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-step", api.Args{"output": "executed"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-pred", pl)
		assert.NoError(t, err)
	})
}

func TestPredicateFalse(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"predicate-false-step", api.ScriptLangAle, "false", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-false-step", api.Args{"output": "should-not-execute"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-false-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-pred-false", pl)
		assert.NoError(t, err)

		assert.False(t, env.MockClient.WasInvoked("predicate-false-step"))
	})
}
func TestLuaPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"lua-pred-step", api.ScriptLangLua, "return true", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"lua-pred-step", api.Args{"output": "executed"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-pred-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-lua-pred", pl)
		assert.NoError(t, err)
	})
}
func TestPredicateError(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"pred-err-step", api.ScriptLangLua,
			"error('predicate failed')", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"pred-err-step", api.Args{"output": "never"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"pred-err-step"},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-pred-err", func() {
			err = env.Engine.StartFlow("wf-pred-err", pl)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.False(t, env.MockClient.WasInvoked("pred-err-step"))
	})
}

func collectPlan(
	sfx string, collect api.InputCollect,
) (*api.Step, *api.Step, *api.Step, *api.ExecutionPlan) {
	providerA := collectProvider(api.StepID("provider-a-" + sfx))
	providerB := collectProvider(api.StepID("provider-b-" + sfx))
	consumer := &api.Step{
		ID:   api.StepID("consumer-" + sfx),
		Name: "Consumer",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {
				Role:     api.RoleRequired,
				Type:     api.TypeAny,
				Required: &api.RequiredConfig{Collect: collect},
			},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
	pl := &api.ExecutionPlan{
		Goals: []api.StepID{consumer.ID},
		Steps: api.Steps{
			providerA.ID: providerA,
			providerB.ID: providerB,
			consumer.ID:  consumer,
		},
		Attributes: api.AttributeGraph{
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
	return providerA, providerB, consumer, pl
}

func collectProvider(id api.StepID) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Provider",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"data": {Role: api.RoleOutput, Type: api.TypeAny},
		},
		HTTP: &api.HTTPConfig{Endpoint: "http://example.com"},
	}
}
