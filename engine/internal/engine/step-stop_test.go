package engine_test

import (
	"testing"
	"time"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
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
			Goals: api.Goals{Steps: []api.StepID{"output-step"}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowCompleted("wf-attr"), func() {
			err = env.Engine.StartPlan("wf-attr", pl)
			testify.NoError(t, err)
		})

		a := assert.New(t)
		a.FlowStateEquals(assert.FlowRef{
			Getter: env.Engine,
			FlowID: "wf-attr",
		}, "test_key", "test_value")
	})
}

func TestMissingOutputFailsFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())
		producer := helpers.NewStepWithOutputs(
			"inventory-resolver", "inventory",
		)
		consumer := helpers.NewSimpleStep("order-creator")
		consumer.Attributes["inventory"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Collect: api.InputCollectFirst,
			},
		}
		testify.NoError(t, env.Engine.RegisterStep(producer))
		testify.NoError(t, env.Engine.RegisterStep(consumer))
		env.MockClient.SetResponse(producer.ID, api.Args{})
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
			Steps: api.Steps{producer.ID: producer, consumer.ID: consumer},
			Attributes: api.AttributeGraph{
				"inventory": {
					Providers: []api.StepID{producer.ID},
					Consumers: []api.StepID{consumer.ID},
				},
			},
		}
		fid := api.FlowID("missing-inventory")
		env.WaitFor(
			wait.StepTerminal(api.FlowStep{FlowID: fid, StepID: producer.ID}),
			func() {
				testify.NoError(t, env.Engine.StartPlan(fid, pl))
			},
		)
		fl, err := env.Engine.GetFlowState(fid)
		testify.NoError(t, err)
		testify.Equal(t, api.FlowFailed, fl.Status)
		testify.Equal(t, api.StepCompleted, fl.Executions[producer.ID].Status)
		for _, work := range fl.Executions[producer.ID].WorkItems {
			testify.Equal(t, api.WorkSucceeded, work.Status)
		}
		ex := fl.Executions[consumer.ID]
		testify.Equal(t, api.StepFailed, ex.Status)
		testify.Equal(t, "required input no longer available", ex.Error)
		testify.False(t, env.MockClient.WasInvoked(consumer.ID))
		testify.NotContains(t, fl.Attributes, api.Name("inventory"))
	})
}

func TestMissingOutputOtherProvider(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())
		first := helpers.NewStepWithOutputs("first-inventory", "inventory")
		other := helpers.NewStepWithOutputs("other-inventory", "inventory")
		other.HTTP.Invoke.Mode = api.ActionModeAsync
		other.HTTP.Invoke.Timeout = time.Minute.Milliseconds()
		consumer := helpers.NewSimpleStep("order-creator")
		consumer.Attributes["inventory"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Collect: api.InputCollectFirst,
			},
		}
		for _, st := range []*api.Step{first, other, consumer} {
			testify.NoError(t, env.Engine.RegisterStep(st))
		}
		env.MockClient.SetResponse(first.ID, api.Args{})
		env.MockClient.SetResponse(other.ID, api.Args{})
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
			Steps: api.Steps{
				first.ID:    first,
				other.ID:    other,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"inventory": {
					Providers: []api.StepID{first.ID, other.ID},
					Consumers: []api.StepID{consumer.ID},
				},
			},
		}
		fid := api.FlowID("other-inventory")
		env.WaitFor(
			wait.StepTerminal(api.FlowStep{FlowID: fid, StepID: first.ID}),
			func() {
				testify.NoError(t, env.Engine.StartPlan(fid, pl))
			},
		)
		fl, err := env.Engine.GetFlowState(fid)
		testify.NoError(t, err)
		testify.Equal(t, api.FlowActive, fl.Status)
		testify.Equal(t, api.StepPending, fl.Executions[consumer.ID].Status)
		testify.False(t, env.MockClient.WasInvoked(consumer.ID))
		fl = env.WaitForFlowStatus(fid, func() {
			for tkn := range fl.Executions[other.ID].WorkItems {
				err := env.Engine.CompleteWork(
					api.FlowStep{FlowID: fid, StepID: other.ID}, tkn,
					api.Args{"inventory": "available"},
				)
				testify.NoError(t, err)
			}
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)
		testify.True(t, env.MockClient.WasInvoked(consumer.ID))
		testify.Equal(t,
			"available", fl.Executions[consumer.ID].Inputs["inventory"])
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
			Goals: api.Goals{Steps: []api.StepID{"step-a", "step-b"}},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}

		fl := env.WaitForFlowStatus("wf-dup-attr", func() {
			err = env.Engine.StartPlan("wf-dup-attr", pl)
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
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"value": {Role: api.RoleOutput, Type: api.TypeString},
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
				"value":  {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
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
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
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
			err := env.Engine.StartPlan("wf-undeclared-outputs", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		testify.NotNil(t, fl.Attributes["value"])
		testify.NotNil(t, fl.Attributes["result"])
		testify.NotContains(t, fl.Attributes, api.Name("extra"))
		testify.NotContains(t, fl.Attributes, api.Name("extra2"))
	})
}

func TestUnconsumedOutputsDiscarded(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		producer := helpers.NewStepWithOutputs("producer", "used", "unused")
		consumer := helpers.NewSimpleStep("consumer")
		consumer.Attributes = api.AttributeSpecs{
			"used":   {Role: api.RoleRequired, Type: api.TypeString},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		}

		testify.NoError(t, env.Engine.RegisterStep(producer))
		testify.NoError(t, env.Engine.RegisterStep(consumer))
		env.MockClient.SetResponse(producer.ID, api.Args{
			"used":   "needed",
			"unused": "discarded",
		})
		env.MockClient.SetResponse(consumer.ID, api.Args{"result": "done"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{consumer.ID}},
			Steps: api.Steps{
				producer.ID: producer,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"used": {
					Providers: []api.StepID{producer.ID},
					Consumers: []api.StepID{consumer.ID},
				},
				"unused": {
					Providers: []api.StepID{producer.ID},
				},
				"result": {
					Providers: []api.StepID{consumer.ID},
				},
			},
		}

		fl := env.WaitForFlowStatus("wf-unconsumed-outputs", func() {
			err := env.Engine.StartPlan("wf-unconsumed-outputs", pl)
			testify.NoError(t, err)
		})

		testify.Equal(t, api.Args{"used": "needed"},
			fl.Executions[producer.ID].Outputs)
		testify.NotContains(t, fl.Attributes, api.Name("unused"))
		testify.Equal(t, api.Args{"result": "done"},
			fl.Executions[consumer.ID].Outputs)
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
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$.payload.value",
						},
					},
				},
			},
		}

		testify.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{
			"payload": map[string]any{"value": "ok"},
		})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-output-mapping")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"input": {"value"}}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		testify.Equal(t, "ok", ex.Outputs["result"])
		testify.Equal(t, "ok", fl.Attributes["result"][0].Value)
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
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{Name: "success"},
				},
			},
		}

		testify.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"success": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-output-rename")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"input": {"test"}}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		testify.Equal(t, "ok", ex.Outputs["status"])
		testify.Equal(t, "ok", fl.Attributes["status"][0].Value)
	})
}
