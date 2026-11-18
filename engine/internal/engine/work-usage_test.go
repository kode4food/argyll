package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	usageToken api.Token  = "token"
	usageStep  api.StepID = "metered"
	usageSpend api.Name   = "spend"
)

func TestCompleteWorkUsageOnce(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		fs := seedUsageWork(t, env, newUsageStep(nil), nil)

		outputs := api.Args{"amount": testCharge}
		err := env.Engine.CompleteWork(fs, usageToken, outputs, workUsage(10))
		assert.NoError(t, err)
		err = env.Engine.CompleteWork(fs, usageToken, outputs, workUsage(10))
		assert.ErrorIs(t, err, api.ErrInvalidWorkTransition)

		work := usageWork(t, env, fs)
		assert.Equal(t, api.WorkSucceeded, work.Status)
		assert.Equal(t, outputs, work.Outputs)
		assert.Equal(t, workUsage(10), work.Usage)
	})
}

func TestWorkUsageSumsAttempts(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		fs := seedUsageWork(t, env, newUsageStep(nil), nil)

		err := env.Engine.RetryWork(fs, usageToken, errors.New("output missing"), workUsage(10))
		assert.NoError(t, err)
		assert.Equal(t, "output missing", usageWork(t, env, fs).Error)

		err = env.Engine.FailWork(fs, usageToken, errors.New("declined"), workUsage(20))
		assert.NoError(t, err)

		work := usageWork(t, env, fs)
		assert.Equal(t, api.WorkFailed, work.Status)
		assert.Equal(t, workUsage(30), work.Usage)
	})
}

func TestWorkUsageMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		f := newMetricFeed(usageSpend, "{}", argExCfg("spend.value"))
		st := newUsageStep(api.AttributeSpecs{
			usageSpend: metricSpec(argExCfg("work.usage")),
		})
		fs := seedUsageWork(t, env, st, f)

		outputs := api.Args{"amount": testCharge}
		err := env.Engine.CompleteWork(fs, usageToken, outputs, workUsage(10))
		assert.NoError(t, err)
		assert.EqualValues(t, map[string]any{
			"input_tokens":  10.0,
			"output_tokens": 20.0,
			"cost":          0.5,
		}, feedTotal(t, env))
	})
}

func TestWorkWithoutUsageMetric(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		charge := newMeasuredStep("charge", api.AttributeSpecs{
			usageSpend: metricSpec(argExCfg("work")),
		})
		err := env.Engine.RegisterStep(charge)
		assert.NoError(t, err)
		err = env.Engine.RegisterFeed(newMetricFeed(usageSpend, "{}", argExCfg("spend.value")))
		assert.NoError(t, err)
		env.MockClient.SetResponse(charge.ID, api.Args{"amount": testCharge})

		runChargeFlow(env, "flow-1", charge.ID)
		assert.NotContains(t, feedTotal(t, env), "usage")
	})
}

// seedUsageWork starts a flow whose one work item awaits its outcome. A non-nil Feed folds the
// step's metrics
func seedUsageWork(
	t *testing.T, env *helpers.TestEngineEnv, st *api.Step, f *api.Feed,
) api.FlowStep {
	t.Helper()
	err := env.Engine.RegisterStep(st)
	assert.NoError(t, err)
	if f != nil {
		err = env.Engine.RegisterFeed(f)
		assert.NoError(t, err)
	}
	fs := api.FlowStep{
		FlowID: "usage-flow",
		StepID: st.ID,
	}
	pl := &api.ExecutionPlan{
		Goals: api.Goals{Steps: []api.StepID{st.ID}},
		Steps: api.Steps{st.ID: st},
		Feeds: usageFeeds(f),
	}
	err = env.SeedStartedWork(fs, pl, usageToken)
	assert.NoError(t, err)
	return fs
}

func newUsageStep(metrics api.AttributeSpecs) *api.Step {
	return &api.Step{
		ID:   usageStep,
		Name: "Metered Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
		},
		WorkConfig: &api.WorkConfig{MaxRetries: 1},
		Attributes: api.Apply(api.AttributeSpecs{
			"amount": {Role: api.RoleOutput},
		}, metrics),
	}
}

func usageFeeds(f *api.Feed) api.PlanFeeds {
	if f == nil {
		return nil
	}
	return api.PlanFeeds{
		f.ID: {
			Feed:    f,
			Sources: api.Slice[api.StepID]{usageStep},
		},
	}
}

func usageWork(t *testing.T, env *helpers.TestEngineEnv, fs api.FlowStep) api.WorkState {
	t.Helper()
	fl, err := env.Engine.GetFlowState(fs.FlowID)
	assert.NoError(t, err)
	return fl.Executions[fs.StepID].WorkItems[usageToken]
}

// workUsage is the usage of an attempt that read n tokens, priced at a twentieth of that
func workUsage(n float64) api.Usage {
	return api.Usage{
		"input_tokens":  n,
		"output_tokens": n * 2,
		"cost":          n / 20,
	}
}
