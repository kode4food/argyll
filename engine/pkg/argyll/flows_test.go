package argyll_test

import (
	"testing"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/argyll"
)

func TestEmbeddedFlow(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{})
	assert.NoError(t, eng.Start())

	st := &api.Step{
		ID:   "embedded-step",
		Name: "Embedded Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return { greeting = "hello" }`,
		},
		Attributes: api.AttributeSpecs{
			"greeting": {Role: api.RoleOutput},
		},
	}
	assert.NoError(t, eng.RegisterStep(st))

	steps, err := eng.ListSteps()
	assert.NoError(t, err)
	assert.Len(t, steps, 1)

	assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
		ID:    "embedded-flow",
		Goals: []api.StepID{st.ID},
	}))

	fl, err := eng.GetFlowState("embedded-flow")
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("embedded-flow"), fl.ID)
}

func TestEmbeddedFlowUnknownGoal(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{})

	err := eng.StartFlow(api.CreateFlowRequest{
		ID:    "embedded-flow",
		Goals: []api.StepID{"nope"},
	})
	assert.ErrorIs(t, err, api.ErrGoalNotFound)
}

func newTestEngine(t *testing.T, opts argyll.Options) argyll.Engine {
	t.Helper()

	opts.Backend = func(pub timebox.Publisher) (timebox.Backend, error) {
		return memory.Open(memory.Config{Publisher: pub}), nil
	}

	eng, err := argyll.New(opts)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = eng.Stop() })
	return eng
}
