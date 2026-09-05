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
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type parentWriteBackend struct {
	timebox.Backend
	beforeAppend func(timebox.AppendRequest) error
}

var (
	ErrParentWrite = errors.New("parent write unavailable")
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
			Goals: []api.StepID{"complete-step"},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-complete", func() {
			err = env.Engine.StartFlow("wf-complete", pl)
			testify.NoError(t, err)
		})
		a.FlowStatus(fl, api.FlowCompleted)
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
			Goals: []api.StepID{"fail-step"},
			Steps: api.Steps{st.ID: st},
		}

		// Wait for flow to fail automatically
		env.WaitFor(wait.FlowFailed("wf-fail"), func() {
			err = env.Engine.StartFlow("wf-fail", pl)
			testify.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState("wf-fail")
		testify.NoError(t, err)
		a.FlowStatus(fl, api.FlowFailed)
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
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
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
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-flow", func() {
			err = env.Engine.StartFlow("parent-flow", pl)
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

			childID := api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-flow", parent.ID, tkn,
			))
			childState, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.Equal(t, api.FlowCompleted, childState.Status)

			testify.Equal(t,
				api.FlowID("parent-flow"), metaFlowID(childState.Metadata),
			)
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
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "held-subflow-step",
			Name: "Held Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
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
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		id := api.FlowID("held-parent-flow")
		env.WaitFor(wait.FlowDeactivated(id), func() {
			testify.NoError(t, env.Engine.StartFlow(id, pl))
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
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		// Parent already deactivated; the child never got its release
		testify.NoError(t, env.RaiseFlowEvents(parentID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: parentID, Plan: pl, Init: api.InitArgs{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowCompleted,
				Data: api.FlowCompletedEvent{FlowID: parentID},
			},
			helpers.FlowEvent{
				Type: api.EventTypeFlowDeactivated,
				Data: api.FlowDeactivatedEvent{
					FlowID: parentID, Status: api.FlowCompleted,
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
				Script:   "error('boom')",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-fail",
			Name: "Subflow Fail",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
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
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-fail", func() {
			err = env.Engine.StartFlow("parent-fail", pl)
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
				Goals: []api.StepID{"missing-goal"},
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
			Goals:    []api.StepID{parent.ID},
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
				Script:   "return {child_out = child_in}",
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
				Goals: []api.StepID{child.ID},
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
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-mapped", func() {
			err = env.Engine.StartFlow("parent-mapped", pl,
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
				Script:   "return {}",
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
				Goals: []api.StepID{child.ID},
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
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		testify.NoError(t, err)

		fl := env.WaitForFlowStatus("parent-missing-output", func() {
			err = env.Engine.StartFlow("parent-missing-output", pl)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, fl.Status)
	})
}

func TestParentNotificationRetries(t *testing.T) {
	var fail atomic.Bool
	checked := make(chan struct{}, 1)
	var envRef *helpers.TestEngineEnv
	parentID := api.FlowID("notification-parent")
	childID := api.FlowID("notification-parent:sub:token")
	backend := &parentWriteBackend{
		Backend: memory.NewPersistence(),
		beforeAppend: func(req timebox.AppendRequest) error {
			if !fail.Load() || !req.ID.Equal(events.FlowKey(parentID)) ||
				len(req.Events) == 0 {
				return nil
			}
			envRef.Engine.ScheduleTask(
				[]string{"notification-checked"}, time.Now(),
				func() error { checked <- struct{}{}; return nil },
			)
			return ErrParentWrite
		},
	}
	cfg := helpers.NewTestConfig()
	store, err := timebox.NewStore(backend, cfg.FlowStoreConfig())
	testify.NoError(t, err)
	helpers.WithTestEnvDeps(t,
		engine.Dependencies{FlowStore: store},
		func(env *helpers.TestEngineEnv) {
			envRef = env
			sub := &api.Step{ID: "sub", Type: api.StepTypeFlow}
			testify.NoError(t,
				env.SeedStartedWork(
					api.FlowStep{FlowID: parentID, StepID: sub.ID},
					&api.ExecutionPlan{
						Goals: []api.StepID{sub.ID},
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
			testify.ErrorIs(t,
				env.Engine.RecoverFlow(childID), ErrParentWrite)
			child, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.True(t, child.DeactivatedAt.IsZero())
			testify.NoError(t, env.Engine.Start())
			<-checked
			// Drain startup reconciliation before allowing writes again.
			barrier := make(chan struct{})
			env.Engine.ScheduleTask(
				[]string{"startup-checked"}, time.Now(),
				func() error { close(barrier); return nil },
			)
			<-barrier
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

func (b *parentWriteBackend) Append(req timebox.AppendRequest) error {
	if err := b.beforeAppend(req); err != nil {
		return err
	}
	return b.Backend.Append(req)
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
