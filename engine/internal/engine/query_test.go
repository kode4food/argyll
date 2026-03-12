package engine_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestQueryFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("test")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"test"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowActivated("wf-list"), func() {
			err = env.Engine.StartFlow("wf-list", pl)
			assert.NoError(t, err)
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{})
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Flows)
	})
}

func TestListFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("list-step")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowActivated("wf-listflows"), func() {
			err = env.Engine.StartFlow("wf-listflows", pl)
			assert.NoError(t, err)
		})

		flows, err := env.Engine.ListFlows()
		assert.NoError(t, err)
		assert.NotEmpty(t, flows)
	})
}

func TestListFlowsIgnoresBadStatusEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addStatusEntry(t, env, events.FlowStatusActive, "bad:flow-id", time.Now())

		flows, err := env.Engine.ListFlows()
		assert.Error(t, err)
		assert.Nil(t, flows)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowStatusEntry))
	})
}

func TestQueryFlowsFiltersAndPaging(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		active := helpers.NewSimpleStep("active-step")
		active.Type = api.StepTypeAsync

		completeStep := helpers.NewSimpleStep("complete-step")
		failed := helpers.NewSimpleStep("failed-step")

		assert.NoError(t, env.Engine.RegisterStep(active))
		assert.NoError(t, env.Engine.RegisterStep(completeStep))
		assert.NoError(t, env.Engine.RegisterStep(failed))

		env.MockClient.SetResponse(completeStep.ID, api.Args{"ok": true})
		env.MockClient.SetError(failed.ID, assert.AnError)

		activePlan := &api.ExecutionPlan{
			Goals: []api.StepID{active.ID},
			Steps: api.Steps{active.ID: active},
		}
		completePlan := &api.ExecutionPlan{
			Goals: []api.StepID{completeStep.ID},
			Steps: api.Steps{completeStep.ID: completeStep},
		}
		failedPlan := &api.ExecutionPlan{
			Goals: []api.StepID{failed.ID},
			Steps: api.Steps{failed.ID: failed},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "flow-active", StepID: active.ID},
			func() {
				assert.NoError(t,
					env.Engine.StartFlow("flow-active", activePlan,
						flow.WithLabels(api.Labels{"tier": "active"}),
					),
				)
			},
		)
		env.WaitForFlowStatus("flow-complete", func() {
			assert.NoError(t,
				env.Engine.StartFlow("flow-complete", completePlan,
					flow.WithLabels(api.Labels{"tier": "done"}),
				),
			)
		})
		env.WaitForFlowStatus("flow-failed", func() {
			assert.NoError(t,
				env.Engine.StartFlow("flow-failed", failedPlan,
					flow.WithLabels(api.Labels{"tier": "fail"}),
				),
			)
		})

		waitForQueryFlow(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowActive},
		}, "flow-active")
		waitForQueryFlow(t, env.Engine, &api.QueryFlowsRequest{
			IDPrefix: "flow-c",
		}, "flow-complete")
		waitForQueryFlow(t, env.Engine, &api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "done"},
		}, "flow-complete")
		waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{}, 3)

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

func TestLabelIndex(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("label-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		labels := api.Labels{
			"tier": "gold",
			"env":  "prod",
		}

		env.WaitForFlowStatus("flow-labeled", func() {
			assert.NoError(t, env.Engine.StartFlow(
				"flow-labeled",
				pl,
				flow.WithLabels(labels),
			))
		})

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
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time { return now },
	}, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("sort-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("flow-a", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-a", pl))
		})
		now = now.Add(10 * time.Millisecond)

		env.WaitForFlowStatus("flow-b", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-b", pl))
		})

		resp := waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		}, 2)
		recent0 := resp.Flows[0].Timestamp
		recent1 := resp.Flows[1].Timestamp
		assert.False(t, recent0.After(recent1))
	})
}

func TestQueryFlowsPaginationAsc(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time { return now },
	}, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("page-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("page-a", func() {
			assert.NoError(t, env.Engine.StartFlow("page-a", pl))
		})
		now = now.Add(10 * time.Millisecond)

		env.WaitForFlowStatus("page-b", func() {
			assert.NoError(t, env.Engine.StartFlow("page-b", pl))
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
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Cursor: "not-base64",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowCursor))
	})
}

func TestQueryFlowsBadCursorJSON(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Cursor: "bm90LWpzb24",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowCursor))
	})
}

func TestQueryFlowsIgnoresBadStatusEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addStatusEntry(t, env, events.FlowStatusActive, "bad:flow-id", time.Now())

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
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addStatusEntry(t, env,
			events.FlowStatusCompleted, "flow:missing-labels", time.Now(),
		)

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
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("no-label-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("flow-no-labels", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-no-labels", pl))
		})

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "done"},
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Flows)
	})
}

func TestLabelIntersection(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("intersection-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("flow-intersection-a", func() {
			assert.NoError(t, env.Engine.StartFlow(
				"flow-intersection-a",
				pl,
				flow.WithLabels(api.Labels{
					"tier": "gold",
					"env":  "prod",
				}),
			))
		})

		env.WaitForFlowStatus("flow-intersection-b", func() {
			assert.NoError(t, env.Engine.StartFlow(
				"flow-intersection-b",
				pl,
				flow.WithLabels(api.Labels{
					"tier": "silver",
					"env":  "stage",
				}),
			))
		})

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

func TestBadLabelEntry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addLabelEntry(t, env, "tier", "gold", "bad:flow-id")

		_, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Labels: api.Labels{"tier": "gold"},
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidFlowLabelEntry))
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

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		var childID api.FlowID
		env.WithConsumer(func(consumer *event.Consumer) {
			parentState := env.WaitForFlowStatus("parent-list", func() {
				err := env.Engine.StartFlow("parent-list", pl)
				assert.NoError(t, err)
			})
			assert.Equal(t, api.FlowCompleted, parentState.Status)

			exec := parentState.Executions[parent.ID]
			if !assert.NotNil(t, exec) {
				return
			}

			var tkn api.Token
			for t := range exec.WorkItems {
				tkn = t
				break
			}

			childID = api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-list", parent.ID, tkn,
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
		for _, fl := range resp.Flows {
			ids = append(ids, fl.ID)
		}

		assert.Contains(t, ids, api.FlowID("parent-list"))
		assert.NotContains(t, ids, childID)
	})
}

func TestQueryFlowsBadStatuses(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("status-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("flow-status", func() {
			assert.NoError(t, env.Engine.StartFlow("flow-status", pl))
		})

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
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time { return now },
	}, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("page-desc-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"ok": true})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForFlowStatus("page-desc-a", func() {
			assert.NoError(t, env.Engine.StartFlow("page-desc-a", pl))
		})
		now = now.Add(10 * time.Millisecond)

		env.WaitForFlowStatus("page-desc-b", func() {
			assert.NoError(t, env.Engine.StartFlow("page-desc-b", pl))
		})

		waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
		}, 2)

		first := waitForQueryFlows(t, env.Engine, &api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentDesc,
			Limit:    1,
		}, 1)
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
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addStatusEntry(t, env, events.FlowStatusCompleted, "flow:flow-b", at)
		addStatusEntry(t, env, events.FlowStatusCompleted, "flow:flow-a", at)

		resp, err := env.Engine.QueryFlows(&api.QueryFlowsRequest{
			Statuses: []api.FlowStatus{api.FlowCompleted},
			Sort:     api.FlowSortRecentAsc,
		})
		assert.NoError(t, err)

		if assert.Len(t, resp.Flows, 2) {
			assert.Equal(t, api.FlowID("flow-a"), resp.Flows[0].ID)
			assert.Equal(t, api.FlowID("flow-b"), resp.Flows[1].ID)
			assert.True(t, resp.Flows[0].Timestamp.Equal(resp.Flows[1].Timestamp))
		}
	})
}

func TestPageTieBreak(t *testing.T) {
	at := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		addStatusEntry(t, env,
			events.FlowStatusCompleted, "flow:page-tie-b", at,
		)
		addStatusEntry(t, env,
			events.FlowStatusCompleted, "flow:page-tie-a", at,
		)

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
	for _, fl := range resp.Flows {
		ids = append(ids, fl.ID)
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

func addStatusEntry(
	t *testing.T, env *helpers.TestEngineEnv, status, id string, at time.Time,
) {
	t.Helper()

	cli := redis.NewClient(&redis.Options{Addr: env.Redis.Addr()})
	defer func() { _ = cli.Close() }()

	err := cli.ZAdd(t.Context(),
		"test-flow:idx:status:"+status,
		redis.Z{Score: float64(at.UnixMilli()), Member: id},
	).Err()
	assert.NoError(t, err)
}

func addLabelEntry(
	t *testing.T, env *helpers.TestEngineEnv, label, value, id string,
) {
	t.Helper()

	cli := redis.NewClient(&redis.Options{Addr: env.Redis.Addr()})
	defer func() { _ = cli.Close() }()

	key := fmt.Sprintf("test-flow:idx:label:%s:%s", label, value)
	err := cli.SAdd(t.Context(), key, id).Err()
	assert.NoError(t, err)
}
