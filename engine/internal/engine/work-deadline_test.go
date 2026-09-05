package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// newDeadlineStep builds an async step whose callback deadline is short enough
// to expire during a test
func newDeadlineStep(id api.StepID) *api.Step {
	st := helpers.NewSimpleStep(id)
	st.HTTP.Invoke.Mode = api.ActionModeAsync
	st.HTTP.Invoke.Timeout = 20
	st.WorkConfig = &api.WorkConfig{
		MaxRetries:  1,
		InitBackoff: 1,
		MaxBackoff:  1,
		BackoffType: api.BackoffTypeFixed,
	}
	return st
}

func TestAsyncCallbackDeadlineExpires(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newDeadlineStep("async-deadline-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		id := api.FlowID("wf-async-deadline")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		// The callback never arrives, so the attempt expires and retries
		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.StartFlow(id, pl))
			w.ForAll(
				wait.WorkNotCompleted(fs),
				wait.WorkRetryScheduled(fs),
			)
		})

		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowFailed, fl.Status)
		for _, work := range fl.Executions[st.ID].WorkItems {
			assert.Equal(t, api.WorkFailed, work.Status)
			assert.Equal(t, 1, work.RetryCount)
		}
	})
}

func TestCallbackBeforeDeadlineSurvives(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("in-time-step")
		st.HTTP.Invoke.Mode = api.ActionModeAsync
		st.HTTP.Invoke.Timeout = 60000
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		id := api.FlowID("wf-in-time")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.WorkStarted(fs), func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		for tkn := range fl.Executions[st.ID].WorkItems {
			assert.NoError(t, env.Engine.CompleteWork(fs, tkn, api.Args{}))
		}

		fl = env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		for _, work := range fl.Executions[st.ID].WorkItems {
			assert.Equal(t, api.WorkSucceeded, work.Status)
			assert.Equal(t, 0, work.RetryCount)
		}
	})
}

func TestOrphanedWorkExpiresOnAnotherNode(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newDeadlineStep("orphan-deadline-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		id := api.FlowID("wf-orphan-deadline")
		tkn := api.Token("work-orphaned")

		// Work claimed by a node that never came back, so no local task for
		// it exists and only this engine's recovery can expire it
		assert.NoError(t, env.SeedStartedWork(
			api.FlowStep{FlowID: id, StepID: st.ID},
			&api.ExecutionPlan{
				Goals: []api.StepID{st.ID},
				Steps: api.Steps{st.ID: st},
			},
			tkn,
		))
		assert.NoError(t, env.Engine.Start())

		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.Equal(t,
			api.WorkFailed, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
	})
}

func TestCompensationDeadlineRetries(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-deadline-step")
		st.HTTP.Compensate.Mode = api.ActionModeAsync
		st.HTTP.Compensate.Timeout = 20
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetCompHandler(st.ID,
			func(client.CompensateRequest) error {
				return nil
			},
		)

		id := api.FlowID("wf-comp-deadline")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-comp-deadline")

		// Compensation is dispatched but never calls back
		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.Start())
			w.ForEvent(wait.CompRetryScheduled(fs))
		})

		fl := helpers.WaitForFlowState(t, env.Engine, helpers.FlowStateQuery{
			FlowID:  id,
			Timeout: wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				work := fl.Executions[st.ID].WorkItems[tkn]
				return work.Status == api.WorkCompFailed
			},
		})
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompFailed, work.Status)
	})
}

func TestMissingChildFlowRecovers(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		child := &api.Step{
			ID:   "orphan-child-step",
			Name: "Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}
		parent := &api.Step{
			ID:         "orphan-parent-step",
			Name:       "Parent Step",
			Type:       api.StepTypeFlow,
			Flow:       &api.FlowConfig{Goals: []api.StepID{child.ID}},
			Attributes: api.AttributeSpecs{},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  1,
				InitBackoff: 1,
				MaxBackoff:  1,
				BackoffType: api.BackoffTypeFixed,
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		id := api.FlowID("wf-missing-child")
		fs := api.FlowStep{FlowID: id, StepID: parent.ID}
		tkn := api.Token("work-no-child")
		assert.NoError(t, env.SeedStartedWork(fs, pl, tkn))

		assert.NoError(t, env.Engine.Start())

		// The recovery deadline of a flow step is the engine step timeout,
		// so allow for a full one to elapse before the retry lands
		fl := helpers.WaitForFlowState(t, env.Engine, helpers.FlowStateQuery{
			FlowID:  id,
			Timeout: 3 * wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				return fl.Status == api.FlowCompleted
			},
		})
		work := fl.Executions[parent.ID].WorkItems[tkn]
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.WorkSucceeded, work.Status)

		childID := api.FlowID(
			string(id) + ":" + string(parent.ID) + ":" + string(tkn),
		)
		childFl, err := env.Engine.GetFlowState(childID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, childFl.Status)
	})
}

// TestLiveChildFlowSurvivesDeadline proves a recovery deadline settles only
// work whose child never launched. A running child keeps its parent in flight
func TestLiveChildFlowSurvivesDeadline(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		child := &api.Step{
			ID:   "live-child-step",
			Name: "Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}
		parent := &api.Step{
			ID:         "live-parent-step",
			Name:       "Parent Step",
			Type:       api.StepTypeFlow,
			Flow:       &api.FlowConfig{Goals: []api.StepID{child.ID}},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		id := api.FlowID("wf-live-child")
		fs := api.FlowStep{FlowID: id, StepID: parent.ID}
		tkn := api.Token("work-live-child")
		assert.NoError(t, env.SeedStartedWork(fs, pl, tkn))

		// The child exists and is still running, so the parent's attempt is
		// accounted for however long the deadline has been past
		childID := api.FlowID(
			string(id) + ":" + string(parent.ID) + ":" + string(tkn),
		)
		assert.NoError(t, env.RaiseFlowEvents(childID, helpers.FlowEvent{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: childID,
				Plan:   pl,
				Init:   api.InitArgs{},
				Metadata: api.Metadata{
					api.MetaParentFlowID:        string(id),
					api.MetaParentStepID:        string(parent.ID),
					api.MetaParentWorkItemToken: string(tkn),
				},
			},
		}))

		cfg := util.MutableCopy(env.Config)
		cfg.StepTimeout = 100
		cfg.Raft.LocalID = "node-live-child"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-live-child", Address: "127.0.0.1:9723"},
		)
		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()
		assert.NoError(t, peer.Start())

		time.Sleep(400 * time.Millisecond)
		fl, err := peer.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkActive, fl.Executions[parent.ID].WorkItems[tkn].Status,
		)
	})
}
