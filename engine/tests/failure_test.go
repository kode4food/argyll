package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestPartialFlowFailure tests that when one step in a flow fails, independent
// steps complete successfully while dependent steps fail
func TestPartialFlowFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: No inputs, produces "valueB" and "valueC"
		stepA := helpers.NewStepWithOutputs("step-a", "valueB", "valueC")

		// Step B: Requires "valueB", will fail
		stepB := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
		stepB.ID = "step-b"
		stepB.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		stepB.Attributes["outputB"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step C: Requires "valueC", should complete successfully
		stepC := helpers.NewTestStepWithArgs([]api.Name{"valueC"}, nil)
		stepC.ID = "step-c"
		stepC.Attributes["outputC"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step D (Goal): Requires both "outputB" and "outputC"
		stepD := helpers.NewTestStepWithArgs(
			[]api.Name{"outputB", "outputC"}, nil,
		)
		stepD.ID = "step-d"
		stepD.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))
		assert.NoError(t, env.Engine.RegisterStep(stepD))

		// Set mock responses
		env.MockClient.SetResponse("step-a", api.Args{
			"valueB": "b-val",
			"valueC": "c-val",
		})
		env.MockClient.SetError("step-b", api.ErrWorkNotCompleted)
		env.MockClient.SetResponse("step-c", api.Args{"outputC": "C-result"})
		env.MockClient.SetResponse("step-d", api.Args{"result": "done"})

		// Create execution plan
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-d"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
				"step-d": stepD,
			},
			Attributes: api.AttributeGraph{
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueC": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-c"},
				},
				"outputB": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-d"},
				},
				"outputC": &api.AttributeEdges{
					Providers: []api.StepID{"step-c"},
					Consumers: []api.StepID{"step-d"},
				},
			},
		}

		flowID := api.FlowID("test-partial-failure")
		env.WaitAfterAll(3, func(waits []*wait.Wait) {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
			waits[0].ForEvent(wait.StepTerminal(api.FlowStep{
				FlowID: flowID,
				StepID: "step-c",
			}))
			waits[1].ForEvent(wait.StepTerminal(api.FlowStep{
				FlowID: flowID,
				StepID: "step-b",
			}))
			waits[2].ForEvent(wait.FlowTerminal(flowID))
		})

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, flow.Status)

		// Verify step A completed (no dependencies, no errors)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)

		// Verify step B failed (configured to fail)
		assert.Equal(t, api.StepFailed, flow.Executions["step-b"].Status)

		// Verify step C completed (independent of B's failure)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-c"].Status)

		// Verify step D failed (depends on B which failed)
		assert.Equal(t, api.StepFailed, flow.Executions["step-d"].Status)

		// Verify attributes from successful steps were set
		assert.Equal(t, "b-val", flow.Attributes["valueB"].Value)
		assert.Equal(t, "c-val", flow.Attributes["valueC"].Value)
		assert.Equal(t, "C-result", flow.Attributes["outputC"].Value)
		assert.NotContains(t, flow.Attributes, "outputB")
		assert.NotContains(t, flow.Attributes, "result")

		// Verify correct steps were invoked
		invocations := env.MockClient.GetInvocations()
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.Contains(t, invocations, api.StepID("step-b"))
		assert.Contains(t, invocations, api.StepID("step-c"))
		assert.NotContains(t, invocations, api.StepID("step-d"))
	})
}

func TestUnreachableStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		stepA := helpers.NewStepWithOutputs("provider-step", "value")

		stepB := helpers.NewTestStepWithArgs([]api.Name{"value"}, nil)
		stepB.ID = "consumer-step"
		stepB.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))

		env.MockClient.SetError(stepA.ID, errors.New("boom"))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{stepB.ID},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
			Attributes: api.AttributeGraph{
				"value": &api.AttributeEdges{
					Providers: []api.StepID{stepA.ID},
					Consumers: []api.StepID{stepB.ID},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{stepB.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flow := env.WaitForFlowStatus("wf-unreachable", func() {
			err := env.Engine.StartFlow("wf-unreachable", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowFailed, flow.Status)

		assert.Equal(t, api.StepFailed, flow.Executions[stepA.ID].Status)
		assert.Equal(t, api.StepFailed, flow.Executions[stepB.ID].Status)
		assert.Equal(t,
			"required input no longer available",
			flow.Executions[stepB.ID].Error,
		)
		assert.NotContains(t, env.MockClient.GetInvocations(), stepB.ID)
	})
}

func TestSkippedProviderCascade(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		orderCreator := helpers.NewStepWithPredicate(
			"order-creator", api.ScriptLangAle, "false", "order",
		)
		orderCreator.Attributes["order"].Type = api.TypeString

		paymentProcessor := helpers.NewTestStepWithArgs(
			[]api.Name{"order"}, nil,
		)
		paymentProcessor.ID = "payment-processor"
		paymentProcessor.Attributes["payment"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		stockReservation := helpers.NewTestStepWithArgs(
			[]api.Name{"order"}, nil,
		)
		stockReservation.ID = "stock-reservation"
		stockReservation.Attributes["reservation"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		notificationSender := helpers.NewTestStepWithArgs(
			[]api.Name{"payment", "reservation"}, nil,
		)
		notificationSender.ID = "notification-sender"
		notificationSender.Attributes["notified"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(orderCreator))
		assert.NoError(t, env.Engine.RegisterStep(paymentProcessor))
		assert.NoError(t, env.Engine.RegisterStep(stockReservation))
		assert.NoError(t, env.Engine.RegisterStep(notificationSender))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{notificationSender.ID},
			Steps: api.Steps{
				orderCreator.ID:       orderCreator,
				paymentProcessor.ID:   paymentProcessor,
				stockReservation.ID:   stockReservation,
				notificationSender.ID: notificationSender,
			},
			Attributes: api.AttributeGraph{
				"order": {
					Providers: []api.StepID{orderCreator.ID},
					Consumers: []api.StepID{
						paymentProcessor.ID,
						stockReservation.ID,
					},
				},
				"payment": {
					Providers: []api.StepID{paymentProcessor.ID},
					Consumers: []api.StepID{notificationSender.ID},
				},
				"reservation": {
					Providers: []api.StepID{stockReservation.ID},
					Consumers: []api.StepID{notificationSender.ID},
				},
				"notified": {
					Providers: []api.StepID{notificationSender.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flow := env.WaitForFlowStatus("wf-skipped-provider", func() {
			err := env.Engine.StartFlow("wf-skipped-provider", plan)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, flow.Status)
		assert.Equal(t,
			api.StepSkipped, flow.Executions[orderCreator.ID].Status,
		)

		for _, stepID := range []api.StepID{
			paymentProcessor.ID,
			stockReservation.ID,
			notificationSender.ID,
		} {
			assert.Equal(t, api.StepFailed, flow.Executions[stepID].Status)
			assert.Equal(t,
				"required input no longer available",
				flow.Executions[stepID].Error,
			)
		}

		invocations := env.MockClient.GetInvocations()
		assert.Empty(t, invocations)
	})
}
