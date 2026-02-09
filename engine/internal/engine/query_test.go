package engine_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type flowEvent struct {
	FlowID api.FlowID `json:"flow_id"`
}

const queryTimeout = 5 * time.Second

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

		consumer := env.EventHub.NewConsumer()

		err = env.Engine.StartFlow("wf-list", plan)
		assert.NoError(t, err)

		helpers.WaitForFlowActivated(t,
			consumer, queryTimeout, "wf-list",
		)

		resp, err := env.Engine.QueryFlows(nil)
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

		consumer := env.EventHub.NewConsumer()

		err = env.Engine.StartFlow("wf-listflows", plan)
		assert.NoError(t, err)

		helpers.WaitForFlowActivated(t,
			consumer, queryTimeout, "wf-listflows",
		)

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
		activeStep.WorkConfig = &api.WorkConfig{MaxRetries: 0}

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

		assert.NoError(t,
			env.Engine.StartFlow("flow-active", activePlan,
				flowopt.WithLabels(api.Labels{"tier": "active"}),
			),
		)
		assert.NoError(t,
			env.Engine.StartFlow("flow-complete", completePlan,
				flowopt.WithLabels(api.Labels{"tier": "done"}),
			),
		)
		assert.NoError(t,
			env.Engine.StartFlow("flow-failed", failedPlan,
				flowopt.WithLabels(api.Labels{"tier": "fail"}),
			),
		)

		env.WaitForStepStarted(t, "flow-active", activeStep.ID, queryTimeout)
		env.WaitForFlowStatus(t, "flow-complete", queryTimeout)
		env.WaitForFlowStatus(t, "flow-failed", queryTimeout)

		waitForQueryFlow(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowActive},
		}, api.FlowID("flow-active"))

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

		assert.NoError(t, env.Engine.StartFlow("flow-a", plan))
		env.WaitForFlowStatus(t, "flow-a", queryTimeout)
		time.Sleep(10 * time.Millisecond)

		assert.NoError(t, env.Engine.StartFlow("flow-b", plan))
		env.WaitForFlowStatus(t, "flow-b", queryTimeout)

		resp := waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		}, 2)
		recent0 := flowRecent(resp.Flows[0].Digest)
		recent1 := flowRecent(resp.Flows[1].Digest)
		assert.False(t, recent0.After(recent1))
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

	deadline := time.Now().Add(queryTimeout)
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

	deadline := time.Now().Add(queryTimeout)
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
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

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

		err := env.Engine.StartFlow("parent-list", plan)
		assert.NoError(t, err)

		parentState := env.WaitForFlowStatus(t, "parent-list", childFlowTimeout)
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

		childID := api.FlowID(fmt.Sprintf(
			"%s:%s:%s", "parent-list", parent.ID, token,
		))

		waitForFlowEvents(t,
			consumer, childFlowTimeout, api.EventTypeFlowActivated,
			"parent-list", childID,
		)
		waitForFlowEvents(t,
			consumer, childFlowTimeout, api.EventTypeFlowDeactivated, childID,
		)

		resp, err := env.Engine.QueryFlows(nil)
		assert.NoError(t, err)

		var ids []api.FlowID
		for _, flow := range resp.Flows {
			ids = append(ids, flow.ID)
		}

		assert.Contains(t, ids, api.FlowID("parent-list"))
		assert.NotContains(t, ids, childID)
	})
}

func waitForFlowEvents(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	typ api.EventType, flowIDs ...api.FlowID,
) {
	t.Helper()

	expected := make(util.Set[api.FlowID], len(flowIDs))
	for _, flowID := range flowIDs {
		expected.Add(flowID)
	}

	filter := helpers.EventDataFilter(
		func(ev *timebox.Event) bool {
			return ev != nil && ev.Type == timebox.EventType(typ)
		},
		func(data flowEvent) bool {
			if expected.Contains(data.FlowID) {
				expected.Remove(data.FlowID)
				return true
			}
			return false
		},
	)

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for len(expected) > 0 {
		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before receiving %d events",
					len(flowIDs),
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for %d events", len(flowIDs))
		}
	}
}
