package engine_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestQueryFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewSimpleStep("test")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"test"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.FlowActivated("wf-list"), func() {
			err = env.Engine.StartFlow("wf-list", plan)
			assert.NoError(t, err)
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Flows)
	})
}

func TestListFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewSimpleStep("list-step")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.FlowActivated("wf-listflows"), func() {
			err = env.Engine.StartFlow("wf-listflows", plan)
			assert.NoError(t, err)
		})

		flows, err := env.Engine.ListFlows()
		assert.NoError(t, err)
		assert.NotEmpty(t, flows)
	})
}

func TestQueryFlowsFiltersAndPaging(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		activeStep := helpers.NewSimpleStep("active-step")
		activeStep.Type = api.StepTypeAsync

		completeStep := helpers.NewSimpleStep("complete-step")
		failedStep := helpers.NewSimpleStep("failed-step")

		assert.NoError(t, env.Engine.RegisterStep(activeStep))
		assert.NoError(t, env.Engine.RegisterStep(completeStep))
		assert.NoError(t, env.Engine.RegisterStep(failedStep))

		env.MockClient.SetResponse(completeStep.ID, api.Args{"ok": true})
		env.MockClient.SetError(failedStep.ID, assert.AnError)

		activePlan := &api.ExecutionPlan{
			Goals: []api.StepID{activeStep.ID},
			Steps: api.Steps{activeStep.ID: activeStep},
		}
		completePlan := &api.ExecutionPlan{
			Goals: []api.StepID{completeStep.ID},
			Steps: api.Steps{completeStep.ID: completeStep},
		}
		failedPlan := &api.ExecutionPlan{
			Goals: []api.StepID{failedStep.ID},
			Steps: api.Steps{failedStep.ID: failedStep},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "flow-active", StepID: activeStep.ID},
			func() {
				assert.NoError(t,
					env.Engine.StartFlow("flow-active", activePlan,
						flowopt.WithLabels(api.Labels{"tier": "active"}),
					),
				)
			},
		)
		env.WaitForFlowStatus("flow-complete", func() {
			assert.NoError(t,
				env.Engine.StartFlow("flow-complete", completePlan,
					flowopt.WithLabels(api.Labels{"tier": "done"}),
				),
			)
		})
		env.WaitForFlowStatus("flow-failed", func() {
			assert.NoError(t,
				env.Engine.StartFlow("flow-failed", failedPlan,
					flowopt.WithLabels(api.Labels{"tier": "fail"}),
				),
			)
		})

		waitForQueryFlow(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowActive},
		}, "flow-active")

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			IDPrefix: "flow-c",
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 1)
		assert.Equal(t, api.FlowID("flow-complete"), resp.Flows[0].ID)

		resp, err = env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "done"},
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 1)
		assert.Equal(t, api.FlowID("flow-complete"), resp.Flows[0].ID)

		resp, err = env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "missing"},
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Flows)

		first, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, first.Count)
		assert.True(t, first.HasMore)
		assert.NotEmpty(t, first.NextCursor)

		second, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Limit:  1,
			Cursor: first.NextCursor,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, second.Count)
		assert.NotEqual(t, first.Flows[0].ID, second.Flows[0].ID)
	})
}

func TestQueryFlowsSortAsc(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewSimpleStep("sort-step")
		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"ok": true})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForFlowStatus("flow-a", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-a", plan))
		})
		time.Sleep(10 * time.Millisecond)

		env.WaitForFlowStatus("flow-b", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-b", plan))
		})

		resp := waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		}, 2)
		recent0 := flowRecent(resp.Flows[0].Digest)
		recent1 := flowRecent(resp.Flows[1].Digest)
		assert.False(t, recent0.After(recent1))
	})
}

func TestQueryFlowsPaginationAsc(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewSimpleStep("page-step")
		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"ok": true})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForFlowStatus("page-a", func() {
			assert.NoError(t, env.Engine.StartFlow("page-a", plan))
		})
		time.Sleep(10 * time.Millisecond)

		env.WaitForFlowStatus("page-b", func() {
			assert.NoError(t, env.Engine.StartFlow("page-b", plan))
		})

		waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
		}, 2)

		first := waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
			Limit:    1,
		}, 1)
		assert.True(t, first.HasMore)
		assert.NotEmpty(t, first.NextCursor)

		second, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
			Limit:    1,
			Cursor:   first.NextCursor,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, second.Count)
		assert.NotEqual(t,
			first.Flows[0].ID, second.Flows[0].ID,
		)
	})
}

func TestQueryFlowsInvalidCursor(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Cursor: "not-base64",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowCursor))
	})
}

func waitForQueryFlow(
	t *testing.T, eng *engine.Engine, req *api.QueryFlowsRequest,
	expected api.FlowID,
) {
	t.Helper()

	deadline := time.Now().Add(wait.DefaultTimeout)
	for time.Now().Before(deadline) {
		resp, err := eng.QueryFlows(req)
		if err == nil &&
			len(resp.Flows) == 1 &&
			resp.Flows[0].ID == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	resp, err := eng.QueryFlows(req)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	var ids []api.FlowID
	for _, flow := range resp.Flows {
		ids = append(ids, flow.ID)
	}
	t.Fatalf("expected flow %s, got %v", expected, ids)
}

func waitForQueryFlows(
	t *testing.T, eng *engine.Engine, req *api.QueryFlowsRequest, min int,
) *api.QueryFlowsResponse {
	t.Helper()

	deadline := time.Now().Add(wait.DefaultTimeout)
	for time.Now().Before(deadline) {
		resp, err := eng.QueryFlows(req)
		if err == nil && len(resp.Flows) >= min {
			return resp
		}
		time.Sleep(10 * time.Millisecond)
	}
	resp, err := eng.QueryFlows(req)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	t.Fatalf("expected at least %d flows, got %d", min, len(resp.Flows))
	return nil
}

func flowRecent(digest *api.FlowDigest) time.Time {
	if digest == nil {
		return time.Time{}
	}
	if digest.Status == api.FlowActive {
		return digest.CreatedAt
	}
	if !digest.CompletedAt.IsZero() {
		return digest.CompletedAt
	}
	return digest.CreatedAt
}

func TestQueryFlowsSkipsChildFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-list",
			Name: "Child List",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-list",
			Name: "Subflow List",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		var childID api.FlowID
		env.WithConsumer(func(consumer *timebox.Consumer) {
			parentState := env.WaitForFlowStatus("parent-list", func() {
				err := env.Engine.StartFlow("parent-list", plan)
				assert.NoError(t, err)
			})
			assert.Equal(t, api.FlowCompleted, parentState.Status)

			exec := parentState.Executions[parent.ID]
			if !assert.NotNil(t, exec) {
				return
			}

			var token api.Token
			for tkn := range exec.WorkItems {
				token = tkn
				break
			}

			childID = api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-list", parent.ID, token,
			))

			w := wait.On(t, consumer)
			w.ForEvents(2,
				wait.FlowActivated("parent-list", childID),
			)
			w.ForEvent(wait.FlowDeactivated(childID))
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)

		var ids []api.FlowID
		for _, flow := range resp.Flows {
			ids = append(ids, flow.ID)
		}

		assert.Contains(t, ids, api.FlowID("parent-list"))
		assert.NotContains(t, ids, childID)
	})
}
