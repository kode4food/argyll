package engine_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
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

		err := env.Engine.StartFlow("wf-defaults", plan, api.Args{
			"input": "value",
		}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-defaults", workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
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

		err := env.Engine.StartFlow(
			"wf-not-complete", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-not-complete", workExecTimeout)
		assert.Equal(t, api.FlowFailed, flow.Status)

		exec := flow.Executions[step.ID]
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

		err := env.Engine.StartFlow("wf-failure", plan, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-failure", workExecTimeout)
		assert.Equal(t, api.FlowFailed, flow.Status)

		exec := flow.Executions[step.ID]
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

		err := env.Engine.StartFlow("wf-meta", plan, api.Args{}, flowMetadata)
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

		err := env.Engine.StartFlow("wf-async-meta", plan, api.Args{}, flowMetadata)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			md := env.MockClient.LastMetadata(step.ID)
			if md == nil {
				return false
			}

			if md["correlation_id"] != "cid-async-123" {
				return false
			}

			webhook, ok := md[api.MetaWebhookURL].(string)
			if !ok {
				return false
			}

			return strings.Contains(webhook, "wf-async-meta") &&
				strings.Contains(webhook, "async-meta")
		}, workExecTimeout, 50*time.Millisecond)
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

		err := env.Engine.StartFlow("wf-script", plan, api.Args{
			"x": float64(2),
		}, api.Metadata{})
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

		err := env.Engine.StartFlow(
			"wf-pred-fail", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			flow, flowErr := env.Engine.GetFlowState("wf-pred-fail")
			if flowErr != nil || flow == nil {
				return false
			}
			exec := flow.Executions[step.ID]
			if exec == nil {
				return false
			}
			return exec.Status == api.StepFailed &&
				strings.Contains(exec.Error, "predicate")
		}, workExecTimeout, 50*time.Millisecond)
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

		err := env.Engine.StartFlow("wf-parallel", plan, api.Args{
			"items": []any{"a", "b", "c"},
		}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, "wf-parallel", workExecTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[step.ID].Status)
	})
}
