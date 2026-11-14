package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestFeedFoldsMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"tries": metricSpec(argExCfg("work.attempts.total")),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("tries", "{}", luaCfg(`
			feed[tries.step] = (feed[tries.step] or 0) + tries.value
			return feed
		`))))
		env.MockClient.SetResponse(charge.ID, api.Args{
			"amount": testCharge,
			"tries":  99,
		})

		runChargeFlow(env, "flow-1", charge.ID)
		runChargeFlow(env, "flow-2", charge.ID)
		assert.EqualValues(t, map[string]any{"charge": 2}, feedTotal(t, env))
	})
}

func TestFeedFoldsMetricsByStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		latency := api.AttributeSpecs{
			"latency": metricSpec(argExCfg("elapsed")),
		}
		a := newMeasuredStep("charge-a", latency)
		b := newMeasuredStep("charge-b", latency)
		assert.NoError(t, env.Engine.RegisterSteps(a, b))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("latency", "{}", argExCfg(`{
			...feed,
			[latency.step]: {
				count: (feed?.[latency.step]?.count ?? 0) + 1,
				total: (feed?.[latency.step]?.total ?? 0) + latency.value,
			},
		}`))))
		env.MockClient.SetResponse(a.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(b.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-a1", a.ID)
		runChargeFlow(env, "flow-a2", a.ID)
		runChargeFlow(env, "flow-b", b.ID)
		res := feedTotal(t, env).(map[string]any)
		assert.EqualValues(t, 2, res["charge-a"].(map[string]any)["count"])
		assert.EqualValues(t, 1, res["charge-b"].(map[string]any)["count"])
	})
}

func TestFeedFoldsDurationMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"took": metricSpec(argExCfg("elapsed")),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("took", "{}", luaCfg(`
			return { measured = took.value >= 0, step = took.step }
		`))))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-1", charge.ID)
		assert.EqualValues(t, map[string]any{
			"measured": true,
			"step":     "charge",
		}, feedTotal(t, env))
	})
}

func TestFeedFoldsFailedMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"ok": metricSpec(argExCfg("succeeded")),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("ok", "0", luaCfg(`
			if ok.value then return feed end
			return feed + 1
		`))))

		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		runChargeFlow(env, "flow-ok", charge.ID)
		assert.EqualValues(t, 0, feedTotal(t, env))

		env.MockClient.SetError(charge.ID, errors.New("boom"))
		fl := runChargeFlow(env, "flow-failed", charge.ID)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.EqualValues(t, 1, feedTotal(t, env))
	})
}

func TestFeedFoldsCompensatedMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		charge := newCompensatedMeasuredStep("charge", api.AttributeSpecs{
			"ok": metricSpec(argExCfg("succeeded")),
		})
		fail := helpers.NewTestStepWithArgs([]api.Name{"amount"}, nil)
		assert.NoError(t, env.Engine.RegisterSteps(charge, fail))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("ok", "0", luaCfg(`
			if ok.value then return feed + 1 end
			return feed
		`))))
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
		work := fl.Executions[charge.ID].WorkItems[firstToken(fl, charge.ID)]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.EqualValues(t, 1, feedTotal(t, env))
	})
}

func TestRegisterFeedAcceptsMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		f := newMetricFeed("tries", "{}", luaCfg("return feed"))
		assert.ErrorIs(t, env.Engine.RegisterFeed(f), api.ErrFeedAttributeUnknown)

		st := newMeasuredStep("charge", api.AttributeSpecs{
			"tries": metricSpec(argExCfg("work.attempts.total")),
		})
		assert.NoError(t, env.Engine.RegisterStep(st))
		assert.NoError(t, env.Engine.RegisterFeed(f))
	})
}

func TestFeedFoldsWholeMetrics(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"work": metricSpec(argExCfg(`{
				ok: succeeded,
				work: work,
			}`)),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(
			newMetricFeed("work", "{}", argExCfg("work.value")),
		))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-1", charge.ID)
		assert.EqualValues(t, map[string]any{
			"ok": true,
			"work": map[string]any{
				"items":  1.0,
				"failed": 0.0,
				"attempts": map[string]any{
					"total":   1.0,
					"average": 1.0,
					"min":     1.0,
					"max":     1.0,
				},
			},
		}, feedTotal(t, env))
	})
}

func TestFeedFoldsLuaMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"tries": metricSpec(luaCfg("return work.attempts.max")),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(
			newMetricFeed("tries", "{}", luaCfg("return tries.value")),
		))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-1", charge.ID)
		assert.EqualValues(t, 1, feedTotal(t, env))
	})
}

func TestFeedOmitsFailedSelect(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"bad": metricSpec(argExCfg("work.missing.field")),
		})
		assert.NoError(t, env.Engine.RegisterStep(charge))
		assert.NoError(t, env.Engine.RegisterFeed(
			newMetricFeed("bad", "0", luaCfg("return 1")),
		))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		fl := runChargeFlow(env, "flow-1", charge.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.EqualValues(t, 0, feedTotal(t, env))
	})
}

func TestDownstreamConsumesMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"tries": metricSpec(argExCfg("work.attempts.total")),
		})
		audit := newConsumerStep("audit", "tries")
		assert.NoError(t, env.Engine.RegisterSteps(charge, audit))
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})
		env.MockClient.SetResponse(audit.ID, api.Args{"handled": true})

		fl := runChargeFlow(env, "flow-audit", audit.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, map[string]any{
			"step":  "charge",
			"value": 1.0,
		}, fl.Executions[audit.ID].Inputs["tries"])
	})
}

func TestFailedStepMetricActivatesConsumer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			"ok": metricSpec(argExCfg("succeeded")),
		})
		recovery := newConsumerStep("recovery", "ok")
		assert.NoError(t, env.Engine.RegisterSteps(charge, recovery))
		env.MockClient.SetError(charge.ID, errors.New("boom"))
		env.MockClient.SetResponse(recovery.ID, api.Args{"handled": true})

		fl := runChargeFlow(env, "flow-recover", recovery.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.StepFailed, fl.Executions[charge.ID].Status)
		assert.Equal(t, map[string]any{
			"step":  "charge",
			"value": false,
		}, fl.Executions[recovery.ID].Inputs["ok"])
	})
}

func TestSkippedStepMetricActivatesConsumer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := &api.Step{
			ID:        "charge",
			Name:      "Skipped Charge",
			Type:      api.StepTypeService,
			Predicate: argExCfg("false"),
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
			Attributes: api.AttributeSpecs{
				"amount": {Role: api.RoleOutput},
				"skip":   metricSpec(argExCfg("skipped")),
			},
		}
		notice := newConsumerStep("notice", "skip")
		assert.NoError(t, env.Engine.RegisterSteps(charge, notice))
		assert.NoError(t, env.Engine.RegisterFeed(newMetricFeed("skip", "0", luaCfg(`
			if skip.value then return feed + 1 end
			return feed
		`))))
		env.MockClient.SetResponse(notice.ID, api.Args{"handled": true})

		fl := runChargeFlow(env, "flow-skip", notice.ID)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.StepSkipped, fl.Executions[charge.ID].Status)
		assert.Equal(t, map[string]any{
			"step":  "charge",
			"value": true,
		}, fl.Executions[notice.ID].Inputs["skip"])
		assert.EqualValues(t, 1, feedTotal(t, env))
	})
}

func TestRegisterStepRejectsBadSelect(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newMeasuredStep("charge", api.AttributeSpecs{
			"tries": metricSpec(argExCfg("work[")),
		})
		assert.Error(t, env.Engine.RegisterStep(st))
	})
}

func metricSpec(sel *api.ScriptConfig) *api.AttributeSpec {
	return &api.AttributeSpec{
		Role:   api.RoleMetric,
		Metric: &api.MetricConfig{Select: sel},
	}
}

// newMeasuredStep builds a Step that outputs amount alongside the given metric Attributes
func newMeasuredStep(id api.StepID, metrics api.AttributeSpecs) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Measured Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
		},
		Attributes: api.Apply(api.AttributeSpecs{
			"amount": {Role: api.RoleOutput},
		}, metrics),
	}
}

// newConsumerStep builds a Step that requires the given Attribute and outputs handled
func newConsumerStep(id api.StepID, input api.Name) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Consumer Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
		},
		Attributes: api.AttributeSpecs{
			input:     {Role: api.RoleRequired},
			"handled": {Role: api.RoleOutput},
		},
	}
}

// newCompensatedMeasuredStep builds a compensated Step whose compensated amount output sits
// alongside the given metric Attributes
func newCompensatedMeasuredStep(id api.StepID, metrics api.AttributeSpecs) *api.Step {
	return &api.Step{
		ID:       id,
		Name:     "Compensated Measured Step",
		Type:     api.StepTypeService,
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080/work"},
			Compensate: &api.HTTPAction{
				Endpoint: "http://test:8080/compensate",
			},
		},
		Attributes: api.Apply(api.AttributeSpecs{
			"amount": {
				Role:        api.RoleOutput,
				Compensated: true,
			},
		}, metrics),
	}
}

func newMetricFeed(name api.Name, seed string, fold *api.ScriptConfig) *api.Feed {
	return &api.Feed{
		ID:         "order-total",
		Name:       "Order Total",
		Value:      seed,
		Fold:       fold,
		Attributes: api.Slice[api.Name]{name},
	}
}
