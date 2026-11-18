package engine_test

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/flow"
	"github.com/kode4food/argyll/engine/pkg/plan"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

type parentWriteBackend struct {
	timebox.Backend
	beforeAppend func(timebox.AppendRequest) error
}

var (
	ErrParentWrite = errors.New("parent write unavailable")

	errLeafDeclined = errors.New("agent declined: unsure")
)

func TestCompleteFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		testify.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithOutputs("complete-step", "result")

		err := env.Engine.RegisterStep(st)
		testify.NoError(t, err)

		// Configure mock to return a result
		env.MockClient.SetResponse("complete-step", api.Args{"result": "final"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"complete-step"}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-complete", func() {
			err = env.Engine.StartPlan("wf-complete", pl)
			testify.NoError(t, err)
		})
		a.FlowStatus(&fl, api.FlowCompleted)
	})
}

func TestFailFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		testify.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("fail-step")

		err := env.Engine.RegisterStep(st)
		testify.NoError(t, err)

		env.MockClient.SetError("fail-step", errors.New("test error"))

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"fail-step"}},
			Steps: api.Steps{st.ID: st},
		}

		// Wait for flow to fail automatically
		env.WaitFor(wait.FlowFailed("wf-fail"), func() {
			err = env.Engine.StartPlan("wf-fail", pl)
			testify.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState("wf-fail")
		testify.NoError(t, err)
		a.FlowStatus(&fl, api.FlowFailed)
		testify.Contains(t, fl.Error, "test error")
	})
}

func TestFlowStepChildSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{},
		}

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-flow", func() {
			err = env.Engine.StartPlan("parent-flow", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[parent.ID]
		if testify.NotNil(t, ex) && testify.NotNil(t, ex.WorkItems) {
			var tkn api.Token
			for t := range ex.WorkItems {
				tkn = t
				break
			}

			childID := api.FlowID(fmt.Sprintf("%s:%s:%s", "parent-flow", parent.ID, tkn))
			childState, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.Equal(t, api.FlowCompleted, childState.Status)

			testify.Equal(t, api.FlowID("parent-flow"), metaFlowID(childState.Metadata))
			testify.Equal(t, parent.ID, metaStepID(childState.Metadata))
			testify.Equal(t, tkn, metaToken(childState.Metadata))
		}
	})
}

func TestChildFlowLease(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "held-child-step",
			Name: "Held Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "held-subflow-step",
			Name: "Held Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{},
		}

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		id := api.FlowID("held-parent-flow")
		env.WaitFor(wait.FlowDeactivated(id), func() {
			testify.NoError(t, env.Engine.StartPlan(id, pl))
		})

		fl, err := env.Engine.GetFlowState(id)
		testify.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[parent.ID].WorkItems {
			tkn = t
			break
		}
		childID := api.FlowID(fmt.Sprintf("%s:%s:%s", id, parent.ID, tkn))

		childFl := helpers.WaitForFlowState(t, env.Engine,
			helpers.FlowStateQuery{
				FlowID:  childID,
				Timeout: time.Second,
				Accept: func(fl api.FlowState) bool {
					return !fl.DeactivatedAt.IsZero()
				},
			})
		testify.False(t, childFl.DeactivatedAt.Before(fl.DeactivatedAt))
	})
}

func TestChildFlowReleaseRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("orphan-child-step")
		parentID := api.FlowID("orphan-parent")
		childID := api.FlowID("orphan-parent:sub:work-a")

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		// Parent already deactivated; the child never got its release
		testify.NoError(t, env.RaiseFlowEvents(parentID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: parentID,
					Plan:   pl,
					Init:   api.InitArgs{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowCompleted,
				Data: api.FlowCompletedEvent{FlowID: parentID},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowDeactivated,
				Data: api.FlowDeactivatedEvent{
					FlowID: parentID,
					Status: api.FlowCompleted,
				},
			},
		))

		testify.NoError(t, env.RaiseFlowEvents(childID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: childID,
					Plan:   pl,
					Init:   api.InitArgs{},
					Metadata: api.Metadata{
						api.MetaParentFlowID:        string(parentID),
						api.MetaParentStepID:        "sub",
						api.MetaParentWorkItemToken: "work-a",
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowCompleted,
				Data: api.FlowCompletedEvent{FlowID: childID},
			},
		))

		fl, err := env.Engine.GetFlowState(childID)
		testify.NoError(t, err)
		testify.True(t, fl.DeactivatedAt.IsZero())

		testify.NoError(t, env.Engine.RecoverFlow(childID))

		fl, err = env.Engine.GetFlowState(childID)
		testify.NoError(t, err)
		testify.False(t, fl.DeactivatedAt.IsZero())
	})
}

func TestFlowStepChildFailureParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-fail",
			Name: "Child Fail",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "error('boom')",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-fail",
			Name: "Subflow Fail",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{},
		}

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-fail", func() {
			err = env.Engine.StartPlan("parent-fail", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, fl.Status)
	})
}

func TestFlowStepMissingGoalParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		parent := &api.Step{
			ID:   "subflow-missing",
			Name: "Subflow Missing",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{"missing-goal"}},
			},
			Attributes: api.AttributeSpecs{},
		}

		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		_, err = plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.Error(t, err)
	})
}

func TestFlowStepMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-mapped",
			Name: "Child Mapped",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "return {child_out = child_in}",
			},
			Attributes: api.AttributeSpecs{
				"child_in":  {Role: api.RoleRequired},
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-mapped",
			Name: "Subflow Mapped",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "child_in"},
					},
				},
				"output": {
					Role: api.RoleOutput,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "child_out"},
					},
				},
			},
		}

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-mapped", func() {
			err = env.Engine.StartPlan("parent-mapped", pl,
				flow.WithInit(api.InitArgs{"input": {float64(7)}}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[parent.ID]
		if testify.NotNil(t, ex) {
			testify.Equal(t, 7, ex.Outputs["output"])
		}
	})
}

func TestFlowStepMissingOutput(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-empty",
			Name: "Child Empty",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "return {}",
			},
			Attributes: api.AttributeSpecs{
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-missing-output",
			Name: "Subflow Missing Output",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{
				"output": {
					Role: api.RoleOutput,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "child_out"},
					},
				},
			},
		}

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		testify.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-missing-output", func() {
			err = env.Engine.StartPlan("parent-missing-output", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, fl.Status)
	})
}

func TestParentNotificationRetries(t *testing.T) {
	var fail atomic.Bool
	checked := make(chan struct{}, 1)
	parentID := api.FlowID("notification-parent")
	childID := api.FlowID("notification-parent:sub:token")
	backend := &parentWriteBackend{
		Backend: memory.Open(),
		beforeAppend: func(req timebox.AppendRequest) error {
			if !fail.Load() || req.ID != events.FlowKey(parentID) ||
				len(req.Events) == 0 {
				return nil
			}
			checked <- struct{}{}
			return ErrParentWrite
		},
	}
	helpers.WithTestBackend(t, backend,
		func(env *helpers.TestEngineEnv) {
			sub := &api.Step{
				ID:   "sub",
				Type: api.StepTypeFlow,
			}
			testify.NoError(t,
				env.SeedStartedWork(
					api.FlowStep{
						FlowID: parentID,
						StepID: sub.ID,
					},
					&api.ExecutionPlan{
						Goals: api.Goals{Steps: []api.StepID{sub.ID}},
						Steps: api.Steps{sub.ID: sub},
					}, "token",
				))
			testify.NoError(t,
				env.RaiseFlowEvents(childID,
					helpers.FlowEvent{
						Type: api.EventTypeFlowStarted,
						Data: api.FlowStartedEvent{
							FlowID: childID,
							Plan:   &api.ExecutionPlan{},
							Metadata: api.Metadata{
								api.MetaParentFlowID:        string(parentID),
								api.MetaParentStepID:        "sub",
								api.MetaParentWorkItemToken: "token",
							},
						},
					},
					helpers.FlowEvent{
						Type: api.EventTypeFlowCompleted,
						Data: api.FlowCompletedEvent{FlowID: childID},
					},
				))
			fail.Store(true)
			testify.ErrorIs(t, env.Engine.RecoverFlow(childID), ErrParentWrite)
			child, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.True(t, child.DeactivatedAt.IsZero())
			testify.NoError(t, env.Engine.Start())
			<-checked
			fail.Store(false)
			fl := env.WaitForTerminalFlow(parentID)
			testify.Equal(t, api.FlowCompleted, fl.Status)
			testify.NoError(t, env.Engine.RecoverFlow(childID))
			child, err = env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.False(t, child.DeactivatedAt.IsZero())
		},
	)
}

func (b *parentWriteBackend) Append(reqs ...timebox.AppendRequest) error {
	for _, req := range reqs {
		if err := b.beforeAppend(req); err != nil {
			return err
		}
	}
	return b.Backend.Append(reqs...)
}

func metaFlowID(meta api.Metadata) api.FlowID {
	switch val := meta[api.MetaParentFlowID].(type) {
	case api.FlowID:
		return val
	case string:
		return api.FlowID(val)
	default:
		return ""
	}
}

func metaStepID(meta api.Metadata) api.StepID {
	switch val := meta[api.MetaParentStepID].(type) {
	case api.StepID:
		return val
	case string:
		return api.StepID(val)
	default:
		return ""
	}
}

func metaToken(meta api.Metadata) api.Token {
	switch val := meta[api.MetaParentWorkItemToken].(type) {
	case api.Token:
		return val
	case string:
		return api.Token(val)
	default:
		return ""
	}
}

// TestParentSettledAtomically proves the child's outcome and the parent's work item now commit
// together. A failing parent write must leave the child unchanged, rather than committing the child
// and losing the notification
func TestParentSettledAtomically(t *testing.T) {
	var fail atomic.Bool
	parentID := api.FlowID("atomic-parent")
	childID := api.FlowID("atomic-parent:sub:token")
	backend := &parentWriteBackend{
		Backend: memory.Open(),
		beforeAppend: func(req timebox.AppendRequest) error {
			if fail.Load() && req.ID == events.FlowKey(parentID) &&
				len(req.Events) != 0 {
				return ErrParentWrite
			}
			return nil
		},
	}
	helpers.WithTestBackend(t, backend,
		func(env *helpers.TestEngineEnv) {
			sub := &api.Step{
				ID:   "sub",
				Type: api.StepTypeFlow,
			}
			testify.NoError(t, env.SeedStartedWork(
				api.FlowStep{
					FlowID: parentID,
					StepID: sub.ID,
				},
				&api.ExecutionPlan{
					Goals: api.Goals{Steps: []api.StepID{sub.ID}},
					Steps: api.Steps{sub.ID: sub},
				}, "token",
			))

			leaf := &api.Step{
				ID:   "leaf",
				Type: api.StepTypeScript,
			}
			testify.NoError(t, env.RaiseFlowEvents(childID,
				helpers.FlowEvent{
					Type: api.EventTypeFlowStarted,
					Data: api.FlowStartedEvent{
						FlowID: childID,
						Plan: &api.ExecutionPlan{
							Goals: api.Goals{Steps: []api.StepID{leaf.ID}},
							Steps: api.Steps{leaf.ID: leaf},
						},
						Init: api.InitArgs{},
						Metadata: api.Metadata{
							api.MetaParentFlowID:        string(parentID),
							api.MetaParentStepID:        "sub",
							api.MetaParentWorkItemToken: "token",
						},
					},
				},
				helpers.FlowEvent{
					Type: api.EventTypeStepStarted,
					Data: api.StepStartedEvent{
						FlowID:    childID,
						StepID:    leaf.ID,
						Inputs:    api.Args{},
						WorkItems: map[api.Token]api.Args{"w": {}},
					},
				},
				helpers.FlowEvent{
					Type: api.EventTypeWorkStarted,
					Data: api.WorkStartedEvent{
						FlowID: childID,
						StepID: leaf.ID,
						Token:  "w",
						Inputs: api.Args{},
					},
				},
			))

			childStep := api.FlowStep{
				FlowID: childID,
				StepID: leaf.ID,
			}
			before, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)

			// the parent write fails, so neither half may land
			fail.Store(true)
			testify.ErrorIs(t,
				env.Engine.CompleteWork(childStep, "w", api.Args{}, nil), ErrParentWrite)

			after, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.Equal(t, before.Status, after.Status,
				"child must not advance when the parent write fails")
			testify.True(t, after.DeactivatedAt.IsZero())

			parent, err := env.Engine.GetFlowState(parentID)
			testify.NoError(t, err)
			testify.True(t, policy.WorkAcceptsResult(
				parent.Executions[sub.ID].WorkItems["token"].Status,
			), "parent work must still be open")

			// with the write allowed, both halves land in one commit
			fail.Store(false)
			testify.NoError(t, env.Engine.CompleteWork(childStep, "w", api.Args{}, nil))

			child, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.Equal(t, api.FlowCompleted, child.Status)

			parent, err = env.Engine.GetFlowState(parentID)
			testify.NoError(t, err)
			testify.Equal(t, api.WorkSucceeded, parent.Executions[sub.ID].WorkItems["token"].Status)
		})
}

func TestFlowStepRollsUpUsage(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		leaf := newAsyncLeafStep("leaf")
		inner := newSubFlowStep("inner", leaf.ID)
		outer := newSubFlowStep("outer", inner.ID)
		err := env.Engine.RegisterSteps(leaf, inner, outer)
		testify.NoError(t, err)

		work := startLeafWork(t, env, "usage-flow", outer.ID)[leaf.ID]
		usage := api.Usage{
			"input_tokens": 10,
			"cost":         0.5,
		}
		fl := env.WaitForFlowStatus("usage-flow", func() {
			testify.NoError(t, completeLeaf(env, work, usage))
		})
		testify.Equal(t, api.FlowCompleted, fl.Status)
		testify.Len(t, fl.Executions[outer.ID].WorkItems, 1)
		for _, item := range fl.Executions[outer.ID].WorkItems {
			testify.Equal(t, usage, item.Usage)
		}
	})
}

func TestFailedFlowStepRollsUpUsage(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		leaf := newAsyncLeafStep("leaf")
		outer := newSubFlowStep("outer", leaf.ID)
		err := env.Engine.RegisterSteps(leaf, outer)
		testify.NoError(t, err)

		work := startLeafWork(t, env, "failed-usage-flow", outer.ID)[leaf.ID]
		usage := api.Usage{"input_tokens": 3}
		fl := env.WaitForFlowStatus("failed-usage-flow", func() {
			testify.NoError(t, failLeaf(env, work, usage))
		})
		testify.Equal(t, api.FlowFailed, fl.Status)
		testify.Len(t, fl.Executions[outer.ID].WorkItems, 1)
		for _, item := range fl.Executions[outer.ID].WorkItems {
			testify.Equal(t, api.WorkFailed, item.Status)
			testify.Equal(t, usage, item.Usage)
		}
	})
}

func TestLateChildUsageReported(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		fast := newAsyncLeafStep("leaf")
		slow := newAsyncLeafStep("leaf-slow")
		outer := &api.Step{
			ID:   "outer",
			Name: "Sub-flow",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{fast.ID, slow.ID}},
			},
			Attributes: api.AttributeSpecs{
				"spend": metricSpec(argExCfg("work.usage")),
			},
		}
		err := env.Engine.RegisterSteps(fast, slow, outer)
		testify.NoError(t, err)
		err = env.Engine.RegisterFeed(newMetricFeed("spend", "{}", argExCfg("spend.value")))
		testify.NoError(t, err)

		works := startLeafWork(t, env, "late-usage-flow", outer.ID)
		env.WaitForFlowStatus("late-usage-flow", func() {
			testify.NoError(t, failLeaf(env, works[fast.ID], api.Usage{"input_tokens": 3}))
		})
		testify.Equal(t, map[string]any{}, feedTotal(t, env))

		env.WaitFor(wait.FlowDeactivated("late-usage-flow"), func() {
			testify.NoError(t, failLeaf(env, works[slow.ID], api.Usage{"input_tokens": 4}))
		})

		fl, err := env.Engine.GetFlowState("late-usage-flow")
		testify.NoError(t, err)
		testify.Len(t, fl.Executions[outer.ID].WorkItems, 1)
		for _, item := range fl.Executions[outer.ID].WorkItems {
			testify.Equal(t, api.WorkFailed, item.Status)
			testify.Equal(t, api.Usage{"input_tokens": 7}, item.Usage)
		}
		testify.EqualValues(t, map[string]any{"input_tokens": 7.0}, feedTotal(t, env))
	})
}

func TestFlowStepMetricsFoldAfterChildDeactivates(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		leaf := newAsyncLeafStep("leaf")
		outer := &api.Step{
			ID:   "outer",
			Name: "Sub-flow",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{leaf.ID}},
			},
			Attributes: api.AttributeSpecs{
				"spend": metricSpec(argExCfg("work.usage")),
			},
		}
		err := env.Engine.RegisterSteps(leaf, outer)
		testify.NoError(t, err)
		err = env.Engine.RegisterFeed(newMetricFeed("spend", "{}", argExCfg("spend.value")))
		testify.NoError(t, err)

		work := startLeafWork(t, env, "fold-flow", outer.ID)[leaf.ID]
		env.WaitFor(wait.FlowDeactivated(work.FlowID), func() {
			testify.NoError(t, completeLeaf(env, work, api.Usage{"input_tokens": 10}))
		})
		testify.EqualValues(t, map[string]any{"input_tokens": 10.0}, feedTotal(t, env))
	})
}

// startLeafWork starts a flow reaching its async leaves through sub-flows and returns each leaf's
// started work by step, waiting for its settlement
func startLeafWork(
	t *testing.T, env *helpers.TestEngineEnv, fid api.FlowID, goal api.StepID,
) map[api.StepID]api.WorkStartedEvent {
	t.Helper()
	res := map[api.StepID]api.WorkStartedEvent{}
	env.WaitForCount(leafCount(env, goal), wait.And(
		wait.Type(api.EventTypeWorkStarted),
		wait.PredicateFilter(func(data api.WorkStartedEvent) bool {
			if !isLeaf(data.StepID) {
				return false
			}
			res[data.StepID] = data
			return true
		}),
	), func() {
		err := env.Engine.StartFlow(api.CreateFlowRequest{
			ID:    fid,
			Goals: api.Goals{Steps: []api.StepID{goal}},
		})
		testify.NoError(t, err)
	})
	return res
}

func completeLeaf(env *helpers.TestEngineEnv, work api.WorkStartedEvent, usage api.Usage) error {
	return env.Engine.CompleteWork(leafStep(work), work.Token, api.Args{}, usage)
}

func failLeaf(env *helpers.TestEngineEnv, work api.WorkStartedEvent, usage api.Usage) error {
	return env.Engine.FailWork(leafStep(work), work.Token, errLeafDeclined, usage)
}

func leafStep(work api.WorkStartedEvent) api.FlowStep {
	return api.FlowStep{
		FlowID: work.FlowID,
		StepID: work.StepID,
	}
}

// leafCount counts the async leaves a goal reaches through its sub-flows
func leafCount(env *helpers.TestEngineEnv, sid api.StepID) int {
	cat, err := env.Engine.GetCatalogState()
	testify.NoError(env.T, err)
	st := cat.Steps[sid]
	if st.Flow == nil {
		return 1
	}
	n := 0
	for _, goal := range st.Flow.Goals.Steps {
		n += leafCount(env, goal)
	}
	return n
}

func isLeaf(sid api.StepID) bool {
	return sid == "leaf" || sid == "leaf-slow"
}

func newAsyncLeafStep(id api.StepID) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Leaf",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://test:8080",
				Mode:     api.ActionModeAsync,
			},
		},
	}
}

func newSubFlowStep(id api.StepID, goals ...api.StepID) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Sub-flow",
		Type: api.StepTypeFlow,
		Flow: &api.FlowConfig{
			Goals: api.Goals{Steps: goals},
		},
	}
}
