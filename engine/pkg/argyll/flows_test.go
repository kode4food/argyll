package argyll_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/kode4food/timebox/raft"
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

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := ln.Addr().String()
	port := ln.Addr().(*net.TCPAddr).Port
	assert.NoError(t, ln.Close())

	nid := "embedded-" + strconv.Itoa(port)
	opts.Raft = raft.Config{
		LocalID: nid,
		Address: addr,
		DataDir: t.TempDir(),
		Servers: []raft.Server{{ID: nid, Address: addr}},
	}

	eng, err := argyll.New(opts)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = eng.Stop() })
	return eng
}
