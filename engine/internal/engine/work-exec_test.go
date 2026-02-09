package engine_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const workExecTimeout = 5 * time.Second

func TestOptionalDefaults(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("default-step")
		step.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role:    api.RoleOptional,
				Type:    api.TypeString,
				Default: "\"fallback\"",
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-defaults")
		workConsumer := env.EventHub.NewConsumer()
		flowConsumer := env.EventHub.NewConsumer()
		err := env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"input": "value"}),
		)
		assert.NoError(t, err)

		helpers.WaitForWorkSucceeded(t,
			workConsumer, flowID, step.ID, 1, workExecTimeout,
		)
		helpers.WaitForFlowCompleted(t, flowConsumer, workExecTimeout, flowID)

		fl, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, "value", exec.Inputs["input"])
		assert.Equal(t, "fallback", exec.Inputs["optional"])
	})
}

func TestIncompleteWorkFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("retry-stop")
		step.WorkConfig = &api.WorkConfig{MaxRetries: 0}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-not-complete")
		workConsumer := env.EventHub.NewConsumer()
		flowConsumer := env.EventHub.NewConsumer()
		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		helpers.WaitForWorkFailed(t,
			workConsumer, flowID, step.ID, 1, workExecTimeout,
		)
		helpers.WaitForFlowFailed(t, flowConsumer, workExecTimeout, flowID)

		fl, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Equal(t, api.ErrWorkNotCompleted.Error(), item.Error)
		}
	})
}

func TestWorkFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("failure-step")
		step.WorkConfig = &api.WorkConfig{MaxRetries: 0}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, errors.New("boom"))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-failure")
		consumer := env.EventHub.NewConsumer()
		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		helpers.WaitForFlowFailed(t, consumer, workExecTimeout, flowID)

		fl, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Contains(t, item.Error, "boom")
		}
	})
}

func TestHTTPMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		flowMetadata := api.Metadata{
			"correlation_id": "cid-123",
			api.MetaFlowID:   "wrong-flow",
			api.MetaStepID:   "wrong-step",
		}

		step := helpers.NewSimpleStep("meta-step")
		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := env.Engine.StartFlow("wf-meta", plan,
			flowopt.WithMetadata(flowMetadata),
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-meta", workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		md := env.MockClient.LastMetadata(step.ID)
		if assert.NotNil(t, md) {
			assert.Equal(t, "cid-123", md["correlation_id"])
			assert.Equal(t, api.FlowID("wf-meta"), md[api.MetaFlowID])
			assert.Equal(t, api.StepID("meta-step"), md[api.MetaStepID])
			assert.NotEmpty(t, md[api.MetaReceiptToken])
			_, hasWebhook := md[api.MetaWebhookURL]
			assert.False(t, hasWebhook)
		}
	})
}

func TestAsyncMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		flowMetadata := api.Metadata{
			"correlation_id": "cid-async-123",
		}

		step := helpers.NewSimpleStep("async-meta")
		step.Type = api.StepTypeAsync
		step.WorkConfig = &api.WorkConfig{MaxRetries: 0}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		consumer := env.EventHub.NewConsumer()
		err := env.Engine.StartFlow("wf-async-meta", plan,
			flowopt.WithMetadata(flowMetadata),
		)
		assert.NoError(t, err)

		helpers.WaitForWorkStarted(t,
			consumer, "wf-async-meta", step.ID, 1, workExecTimeout,
		)

		assert.True(t, env.MockClient.WaitForInvocation(
			step.ID, workExecTimeout,
		))

		md := env.MockClient.LastMetadata(step.ID)
		assert.NotNil(t, md)
		assert.Equal(t, "cid-async-123", md["correlation_id"])

		webhook, ok := md[api.MetaWebhookURL].(string)
		assert.True(t, ok)
		assert.True(t, strings.Contains(webhook, "wf-async-meta"))
		assert.True(t, strings.Contains(webhook, "async-meta"))
	})
}

func TestScriptWorkExecutes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := &api.Step{
			ID:   "script-work",
			Name: "Script Work",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { result = (x or 0) * 3 }",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := env.Engine.StartFlow("wf-script", plan,
			flowopt.WithInit(api.Args{"x": float64(2)}),
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-script", workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		assert.Equal(t, float64(6), exec.Outputs["result"])
	})
}

func TestPredicateFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithPredicate(
			"pred-fail", api.ScriptLangLua, "error('boom')",
		)

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := env.Engine.StartFlow("wf-pred-fail", plan)
		assert.NoError(t, err)

		exec := env.WaitForStepStatus(t,
			"wf-pred-fail", step.ID, workExecTimeout,
		)
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.True(t, strings.Contains(exec.Error, "predicate"))
	})
}

func TestParallelWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "parallel-items"
		step.WorkConfig = &api.WorkConfig{Parallelism: 2}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := env.Engine.StartFlow("wf-parallel", plan,
			flowopt.WithInit(api.Args{"items": []any{"a", "b", "c"}}),
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-parallel", workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[step.ID].Status)
	})
}

func TestRetryPendingParallelism(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "retry-parallel"
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			Backoff:     500,
			MaxBackoff:  500,
			BackoffType: api.BackoffTypeFixed,
			Parallelism: 1,
		}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-retry-parallel")
		workConsumer := env.EventHub.NewConsumer()
		err := env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"items": []any{"a", "b"}}),
		)
		assert.NoError(t, err)

		helpers.WaitForWorkRetryScheduled(t,
			workConsumer, flowID, step.ID, 2, workExecTimeout,
		)

		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		flow := env.WaitForFlowStatus(t, flowID, workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		assert.Len(t, exec.WorkItems, 2)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}

func TestPredicateFailurePerWorkItem(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "predicate-items"
		step.Predicate = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: "if type(items) ~= 'table' then error('boom') end; " +
				"return true",
		}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-pred-work-item")
		err := env.Engine.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"items": []any{"a", "b"}}),
		)
		assert.NoError(t, err)

		exec := env.WaitForStepStatus(t, flowID, step.ID, workExecTimeout)
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Contains(t, exec.Error, "predicate")

		flow := env.WaitForFlowStatus(t, flowID, workExecTimeout)
		assert.Equal(t, api.FlowFailed, flow.Status)
	})
}
