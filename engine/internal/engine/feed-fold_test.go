package engine_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

const testCharge = 5.0

func newChargeStep(id api.StepID) *api.Step {
	st := helpers.NewStepWithOutputs(id, "amount")
	st.Attributes["amount"].Type = api.TypeNumber
	return st
}

func newCompensatedChargeStep(id api.StepID) *api.Step {
	st := newCompensatingStep(id)
	st.Attributes["amount"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeNumber,
	}
	return st
}

func feedTotal(t *testing.T, env *helpers.TestEngineEnv) any {
	t.Helper()
	res, err := env.Engine.GetFeed("order-total")
	assert.NoError(t, err)
	return res.State.Value
}

func runChargeFlow(
	env *helpers.TestEngineEnv, id api.FlowID, sid api.StepID,
) api.FlowState {
	return env.WaitForFlowStatus(id, func() {
		err := env.Engine.StartFlow(api.CreateFlowRequest{
			ID:    id,
			Goals: api.Goals{Steps: []api.StepID{sid}},
		})
		assert.NoError(env.T, err)
	})
}

func TestFeedFoldsOnCompletion(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		fl := runChargeFlow(env, "flow-1", charge.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.EqualValues(t, testCharge, feedTotal(t, env))

		runChargeFlow(env, "flow-2", charge.ID)
		assert.EqualValues(t, 2*testCharge, feedTotal(t, env))
	})
}

func TestFeedFoldsMemoizedResult(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		charge.Handling = api.HandlingMemoized
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-1", charge.ID)
		fl := runChargeFlow(env, "flow-2", charge.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Len(t, env.MockClient.GetInvocations(), 1)
		assert.EqualValues(t, 2*testCharge, feedTotal(t, env))
	})
}

func TestFeedFoldsEveryProvider(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))

		// registered after the Feed, and contributes without changing it
		refund := newChargeStep("refund")
		assert.NoError(t, env.Engine.RegisterStep(refund))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(refund.ID, api.Args{"amount": -1.0})

		runChargeFlow(env, "charge-flow", charge.ID)
		runChargeFlow(env, "refund-flow", refund.ID)
		assert.EqualValues(t, testCharge-1, feedTotal(t, env))
	})
}

func TestFeedFoldsEachAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		refund := helpers.NewStepWithOutputs("refund", "refunded")
		assert.NoError(t, env.Engine.RegisterSteps(charge, refund))
		f := newTestFeed()
		f.Attributes = api.Slice[api.Name]{"amount", "refunded"}
		f.Fold.Script = "return feed + (amount or 0) - (refunded or 0)"
		assert.NoError(t, env.Engine.RegisterFeed(f))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(refund.ID, api.Args{"refunded": 2.0})

		runChargeFlow(env, "charge-flow", charge.ID)
		runChargeFlow(env, "refund-flow", refund.ID)
		assert.EqualValues(t, testCharge-2, feedTotal(t, env))
	})
}

func TestFeedFoldsOnlyItsSpace(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		sp := api.Space{
			ID:   "billing",
			Name: "Billing",
			QBE:  api.SpaceQuery{{"domain:billing"}},
		}
		assert.NoError(t, env.Engine.RegisterSpace(sp))
		charge := newChargeStep("charge")
		charge.Tags = api.Tags{"domain:billing"}
		other := newChargeStep("other")
		assert.NoError(t, env.Engine.RegisterSteps(charge, other))
		f := newTestFeed()
		f.SpaceID = "billing"
		assert.NoError(t, env.Engine.RegisterFeed(f))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(other.ID, api.Args{"amount": 100.0})

		runChargeFlow(env, "charge-flow", charge.ID)
		runChargeFlow(env, "other-flow", other.ID)
		assert.EqualValues(t, testCharge, feedTotal(t, env))
	})
}

func TestFeedFoldIgnoresUnconsumedOutput(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		charge.Attributes["tag"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}
		reader := helpers.NewStepWithOutputs("reader", "ran")
		reader.Attributes["tag"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		assert.NoError(t, env.Engine.RegisterSteps(charge, reader))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{
			"amount": testCharge,
			"tag":    "t",
		})
		env.MockClient.SetResponse(reader.ID, api.Args{"ran": "yes"})

		// Nothing in this Flow reads amount, so the Flow drops it. The Feed
		// must still fold what the Step produced
		fl := runChargeFlow(env, "flow-unconsumed", reader.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.NotContains(t, fl.GetAttributes(), api.Name("amount"))
		assert.EqualValues(t, testCharge, feedTotal(t, env))
	})
}

func TestFeedFoldsCompensatedStepAtDeactivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		charge := newCompensatedChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			err := env.Engine.StartFlow(api.CreateFlowRequest{
				ID:    "flow-comp",
				Goals: api.Goals{Steps: []api.StepID{charge.ID}},
			})
			assert.NoError(t, err)
			w.ForEvent(wait.FlowDeactivated("flow-comp"))
		})
		fl, err := env.Engine.GetFlowState("flow-comp")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.EqualValues(t, testCharge, feedTotal(t, env))
	})
}

func TestFeedSkipsCompensatedWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		charge := newCompensatedChargeStep("charge")
		charge.Attributes["amount"].Compensated = true
		fail := helpers.NewSimpleStep("fail")
		fail.Attributes["amount"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeNumber,
		}
		fail.Attributes["done"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}
		assert.NoError(t, env.Engine.RegisterSteps(charge, fail))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetError(fail.ID, errors.New("boom"))

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			err := env.Engine.StartFlow(api.CreateFlowRequest{
				ID:    "flow-rollback",
				Goals: api.Goals{Steps: []api.StepID{fail.ID}},
			})
			assert.NoError(t, err)
			w.ForEvent(wait.FlowDeactivated("flow-rollback"))
		})
		fl, err := env.Engine.GetFlowState("flow-rollback")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		work := fl.Executions[charge.ID].WorkItems[firstToken(fl, charge.ID)]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.EqualValues(t, 0, feedTotal(t, env))
	})
}

func TestFeedIgnoresFailedStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetError(charge.ID, errors.New("boom"))

		fl := runChargeFlow(env, "flow-failed", charge.ID)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.EqualValues(t, 0, feedTotal(t, env))
	})
}

func TestFeedFoldFailureKeepsValue(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		f := newTestFeed()
		f.Value = `{"charges":["initial"]}`
		f.Fold.Script = `
			feed.charges[1] = amount < 0 and "negative" or "positive"
			if amount < 0 then error("negative charge") end
			return feed
		`
		assert.NoError(t, env.Engine.RegisterFeed(f))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-good-fold", charge.ID)
		want := map[string]any{"charges": []any{"positive"}}
		require.Equal(t, want, feedTotal(t, env))

		env.MockClient.SetResponse(charge.ID, api.Args{"amount": -testCharge})
		fl := runChargeFlow(env, "flow-bad-fold", charge.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, want, feedTotal(t, env))
	})
}

func TestFeedNonFiniteResult(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newChargeStep("charge")
		require.NoError(t, env.Engine.RegisterStep(st))
		f := newTestFeed()
		f.Fold.Script = "return 0/0"
		require.NoError(t, env.Engine.RegisterFeed(f))
		env.MockClient.SetResponse(st.ID, api.Args{"amount": testCharge})

		fl := runChargeFlow(env, "bad-result", st.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		key := events.FeedKey(f.ID, f.Version())
		state, seq, err := env.Engine.GetFeedStateSeq(key)
		require.NoError(t, err)
		assert.EqualValues(t, 1, seq)
		assert.Empty(t, state.Version)
		assert.EqualValues(t, 0, feedTotal(t, env))
	})
}

func TestFeedFoldsMatchingOutputsTogether(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewStepWithOutputs("source", "a", "b")
		require.NoError(t, env.Engine.RegisterStep(st))
		f := newTestFeed()
		f.Attributes = api.Slice[api.Name]{"b", "a"}
		f.Value = `{"pair":"left:right","count":1}`
		f.Fold.Script = `return {pair = a .. ":" .. b, count = 1}`
		require.NoError(t, env.Engine.RegisterFeed(f))
		key := events.FeedKey(f.ID, f.Version())
		run := func(id api.FlowID, a string, wantSeq int64) api.FeedState {
			env.MockClient.SetResponse(st.ID, api.Args{"a": a, "b": "right"})
			runChargeFlow(env, id, st.ID)
			state, seq, err := env.Engine.GetFeedStateSeq(key)
			require.NoError(t, err)
			require.Equal(t, wantSeq, seq)
			return state
		}
		run("matches-seed", "left", 0)
		changed := run("changes-pair", "new", 1)
		require.Equal(t, "new:right", changed.Value.(map[string]any)["pair"])
		assert.Equal(t, changed, run("matches-current", "new", 1))
	})
}

func TestFeedSnapshotGatesPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		gate := helpers.NewStepWithPredicate(
			"gate", api.ScriptLangLua, "return total < 5", "ran",
		)
		gate.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "order-total"},
		}
		assert.NoError(t, env.Engine.RegisterStep(gate))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(gate.ID, api.Args{"ran": "yes"})

		before := runChargeFlow(env, "gate-before", gate.ID)
		assert.Equal(t, api.FlowCompleted, before.Status)
		assert.EqualValues(t, 0, before.Feeds["order-total"])
		assert.Equal(t, api.StepCompleted, before.Executions[gate.ID].Status)

		runChargeFlow(env, "charge-flow", charge.ID)

		after := runChargeFlow(env, "gate-after", gate.ID)
		assert.EqualValues(t, testCharge, after.Feeds["order-total"])
		assert.Equal(t, api.StepSkipped, after.Executions[gate.ID].Status)

		// the first Flow's snapshot is unchanged by later folds
		again, err := env.Engine.GetFlowState("gate-before")
		assert.NoError(t, err)
		assert.EqualValues(t, 0, again.Feeds["order-total"])
	})
}

func TestFeedChildFlowSnapshot(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		reader := helpers.NewScriptStep(
			"child-reader", api.ScriptLangLua, "return {seen = total}", "seen",
		)
		reader.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "order-total"},
		}
		sub := &api.Step{
			ID:   "sub",
			Name: "Sub",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{reader.ID}},
			},
			Attributes: api.AttributeSpecs{
				"amount": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}
		peek := helpers.NewStepWithOutputs("peek", "ran")
		peek.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "order-total"},
		}
		assert.NoError(t, env.Engine.RegisterSteps(reader, sub, peek))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(peek.ID, api.Args{"ran": "yes"})

		// The child starts after charge folds, so its snapshot is newer than
		// the one its parent took when it started
		fl := env.WaitForFlowStatus("parent-feed", func() {
			err := env.Engine.StartFlow(api.CreateFlowRequest{
				ID:    "parent-feed",
				Goals: api.Goals{Steps: []api.StepID{sub.ID, peek.ID}},
			})
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.EqualValues(t, 0, fl.Feeds["order-total"])

		child, err := env.Engine.GetFlowState(
			api.FlowID("parent-feed:sub:" + string(firstToken(fl, sub.ID))),
		)
		assert.NoError(t, err)
		assert.EqualValues(t, testCharge, child.Feeds["order-total"])
	})
}

func TestFeedSurvivesRestart(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		runChargeFlow(env, "before-restart", charge.ID)
		assert.NoError(t, env.Engine.Stop())

		eng, err := env.NewEngineInstance()
		assert.NoError(t, err)
		res, err := eng.GetFeed("order-total")
		assert.NoError(t, err)
		assert.EqualValues(t, testCharge, res.State.Value)
		assert.NoError(t, eng.Stop())
	})
}

func TestFeedConcurrentFolds(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newChargeStep("charge")
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newTestFeed()))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		const n = 8
		ids := make([]api.FlowID, n)
		for i := range ids {
			ids[i] = api.FlowID(fmt.Sprintf("concurrent-%d", i))
		}
		var wg sync.WaitGroup
		for _, id := range ids {
			wg.Go(func() {
				err := env.Engine.StartFlow(api.CreateFlowRequest{
					ID:    id,
					Goals: api.Goals{Steps: []api.StepID{charge.ID}},
				})
				assert.NoError(t, err)
			})
		}
		wg.Wait()
		env.WaitForTerminalFlows(ids, wait.DefaultTimeout)
		assert.EqualValues(t, n*testCharge, feedTotal(t, env))
	})
}

func firstToken(fl api.FlowState, sid api.StepID) api.Token {
	for tkn := range fl.Executions[sid].WorkItems {
		return tkn
	}
	return ""
}
