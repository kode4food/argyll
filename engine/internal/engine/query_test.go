package engine_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestQueryFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow("wf-list", api.FlowActive, nil))

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Flows)
	})
}

func TestListFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow("wf-listflows", api.FlowActive, nil))

		flows, err := env.Engine.ListFlows()
		assert.NoError(t, err)
		assert.NotEmpty(t, flows)
	})
}

func TestListFlowsIgnoresBadStatusEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusActive,
			prefix: "bad",
			id:     "flow-id",
			at:     scheduler.Now(),
		})

		flows, err := env.Engine.ListFlows()
		assert.Error(t, err)
		assert.Nil(t, flows)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowStatusEntry))
	})
}

func TestQueryFlowsFiltersAndPaging(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow(
			"flow-active", api.FlowActive, api.Labels{"tier": "active"},
		))
		assert.NoError(t, env.SeedFlow(
			"flow-complete", api.FlowCompleted, api.Labels{"tier": "done"},
		))
		assert.NoError(t, env.SeedFlow(
			"flow-failed", api.FlowFailed, api.Labels{"tier": "fail"},
		))

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowActive},
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 1)
		assert.Equal(t, api.FlowID("flow-active"), resp.Flows[0].ID)

		resp, err = env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 3)

		resp, err = env.Engine.QueryFlows(&api.QueryFlowsRequest{
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

func TestLabelIndex(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		labels := api.Labels{
			"tier": "gold",
			"env":  "prod",
		}
		assert.NoError(t, env.SeedFlow(
			"flow-labeled", api.FlowCompleted, labels,
		))

		ids, err := env.ListFlowsByLabel("tier", "gold")
		assert.NoError(t, err)
		assert.Contains(t, ids, events.FlowKey("flow-labeled"))

		ids, err = env.ListFlowsByLabel("env", "prod")
		assert.NoError(t, err)
		assert.Contains(t, ids, events.FlowKey("flow-labeled"))

		ids, err = env.ListFlowsByLabel("tier", "silver")
		assert.NoError(t, err)
		assert.NotContains(t, ids, events.FlowKey("flow-labeled"))
	})
}

func TestQueryFlowsSortAsc(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "flow-a",
			at:     now,
		})
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "flow-b",
			at:     now.Add(10 * time.Millisecond),
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 2)
		recent0 := resp.Flows[0].Timestamp
		recent1 := resp.Flows[1].Timestamp
		assert.False(t, recent0.After(recent1))
	})
}

func TestQueryFlowsPaginationAsc(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-a",
			at:     now,
		})
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-b",
			at:     now.Add(10 * time.Millisecond),
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 2)

		first, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
			Limit:    1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, first.Count)
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
		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Cursor: "not-base64",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowCursor))
	})
}

func TestQueryFlowsBadCursorJSON(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Cursor: "bm90LWpzb24",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowCursor))
	})
}

func TestQueryFlowsIgnoresBadStatusEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusActive,
			prefix: "bad",
			id:     "flow-id",
			at:     scheduler.Now(),
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowActive},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowStatusEntry))
	})
}

func TestQueryFlowsStaleLabels(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "missing-labels",
			at:     scheduler.Now(),
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Labels:   api.Labels{"tier": "done"},
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Flows)
	})
}

func TestQueryFlowsNoLabels(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow(
			"flow-no-labels", api.FlowCompleted, nil,
		))

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "done"},
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Flows)
	})
}

func TestLabelIntersection(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow(
			"flow-intersection-a", api.FlowCompleted,
			api.Labels{"tier": "gold", "env": "prod"},
		))
		assert.NoError(t, env.SeedFlow(
			"flow-intersection-b", api.FlowCompleted,
			api.Labels{"tier": "silver", "env": "stage"},
		))

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{
				"tier": "gold",
				"env":  "stage",
			},
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Flows)
	})
}

func TestBadIndexedFlowEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addLabelEntry(t, addLabelEntryArgs{
			env:    env,
			label:  "tier",
			value:  "gold",
			prefix: "bad",
			id:     "flow-id",
		})

		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "gold"},
		})
		assert.Error(t, err)
		assert.True(t,
			errors.Is(err, engine.ErrInvalidFlowStatusEntry) ||
				errors.Is(err, engine.ErrInvalidFlowLabelEntry),
		)
	})
}

func TestSkipChildFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

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

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Catalog:  cat,
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		var childID api.FlowID
		fl := env.WaitForFlowStatus("parent-list", func() {
			err = env.Engine.StartFlow("parent-list", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[parent.ID]
		if !assert.NotNil(t, ex) {
			return
		}

		var tkn api.Token
		for t := range ex.WorkItems {
			tkn = t
			break
		}

		childID = api.FlowID(fmt.Sprintf(
			"%s:%s:%s", "parent-list", parent.ID, tkn,
		))

		childFlow := helpers.WaitForFlowExists(t, env.Engine, childID)
		assert.Equal(t, api.FlowCompleted, childFlow.Status)

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)

		var ids []api.FlowID
		for _, fl := range resp.Flows {
			ids = append(ids, fl.ID)
		}

		assert.Contains(t, ids, api.FlowID("parent-list"))
		assert.NotContains(t, ids, childID)
	})
}

func TestQueryFlowsBadStatuses(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.SeedFlow(
			"flow-status", api.FlowCompleted, nil,
		))

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{
				api.FlowCompleted, api.FlowCompleted, "bogus",
			},
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 1)
		assert.Equal(t, api.FlowID("flow-status"), resp.Flows[0].ID)
	})
}

func TestQueryFlowsPageDesc(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-desc-a",
			at:     now,
		})
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-desc-b",
			at:     now.Add(10 * time.Millisecond),
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Flows, 2)

		first, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentDesc,
			Limit:    1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, first.Count)
		assert.True(t, first.HasMore)
		assert.NotEmpty(t, first.NextCursor)

		second, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentDesc,
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

func TestSortTieBreak(t *testing.T) {
	at := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "flow-b",
			at:     at,
		})
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "flow-a",
			at:     at,
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		})
		assert.NoError(t, err)

		if assert.Len(t, resp.Flows, 2) {
			assert.Equal(t, api.FlowID("flow-a"), resp.Flows[0].ID)
			assert.Equal(t, api.FlowID("flow-b"), resp.Flows[1].ID)
			assert.True(t,
				resp.Flows[0].Timestamp.Equal(resp.Flows[1].Timestamp),
			)
		}
	})
}

func TestPageTieBreak(t *testing.T) {
	at := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-tie-b",
			at:     at,
		})
		addStatusEntry(t, addStatusEntryArgs{
			env:    env,
			status: events.FlowStatusCompleted,
			prefix: "flow",
			id:     "page-tie-a",
			at:     at,
		})

		first, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
			Limit:    1,
		})
		assert.NoError(t, err)
		assert.True(t, first.HasMore)
		assert.NotEmpty(t, first.NextCursor)
		assert.Equal(t, api.FlowID("page-tie-a"), first.Flows[0].ID)

		second, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
			Limit:    1,
			Cursor:   first.NextCursor,
		})
		assert.NoError(t, err)
		assert.Len(t, second.Flows, 1)
		assert.Equal(t, api.FlowID("page-tie-b"), second.Flows[0].ID)
	})
}

type addStatusEntryArgs struct {
	env    *helpers.TestEngineEnv
	status string
	prefix string
	id     string
	at     time.Time
}

func addStatusEntry(t *testing.T, args addStatusEntryArgs) {
	t.Helper()

	aggID := timebox.NewAggregateID(
		timebox.ID(args.prefix), timebox.ID(args.id),
	)
	raw, err := marshalIndexedFlowEvent(args.status, nil)
	assert.NoError(t, err)
	err = args.env.AppendEvents(aggID, 0, &timebox.Event{
		AggregateID: aggID,
		Timestamp:   args.at,
		Type:        indexEventType(args.status),
		Data:        raw,
	})
	assert.NoError(t, err)
}

type addLabelEntryArgs struct {
	env    *helpers.TestEngineEnv
	label  string
	value  string
	prefix string
	id     string
}

func addLabelEntry(t *testing.T, args addLabelEntryArgs) {
	t.Helper()

	aggID := timebox.NewAggregateID(
		timebox.ID(args.prefix), timebox.ID(args.id),
	)
	raw, err := marshalIndexedFlowEvent(
		events.FlowStatusActive,
		api.Labels{args.label: args.value},
	)
	assert.NoError(t, err)
	err = args.env.AppendEvents(aggID, 0, &timebox.Event{
		AggregateID: aggID,
		Timestamp:   scheduler.Now().UTC(),
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        raw,
	})
	assert.NoError(t, err)
}

func marshalIndexedFlowEvent(status string, labels api.Labels) ([]byte, error) {
	if status == events.FlowStatusActive {
		return json.Marshal(api.FlowStartedEvent{
			FlowID: "fixture",
			Labels: labels,
		})
	}
	return json.Marshal(api.FlowDeactivatedEvent{
		FlowID: "fixture",
		Status: api.FlowStatus(status),
	})
}

func indexEventType(status string) timebox.EventType {
	if status == events.FlowStatusActive {
		return timebox.EventType(api.EventTypeFlowStarted)
	}
	return timebox.EventType(api.EventTypeFlowDeactivated)
}
