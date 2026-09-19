package argyll_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/argyll"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

const localStepType api.StepType = "local"

func TestEmbeddedStepType(t *testing.T) {
	ran := make(chan api.StepID, 1)
	eng := newTestEngine(t, argyll.Options{
		Handlers: step.Handlers{
			api.StepTypeScript: builtins.Script(),
			localStepType: {
				Invoke: func(
					rt step.Runtime, st *api.Step,
					inputs api.Args, tkn api.Token,
				) error {
					ran <- st.ID
					return rt.CompleteWork(tkn, api.Args{
						"greeting": "hello",
					})
				},
			},
		},
	})
	assert.NoError(t, eng.Start())

	st := &api.Step{
		ID:   "local-step",
		Name: "Local Step",
		Type: localStepType,
		Attributes: api.AttributeSpecs{
			"greeting": {Role: api.RoleOutput},
		},
	}
	assert.NoError(t, eng.RegisterStep(st))
	assert.NoError(t, eng.RegisterStep(&api.Step{
		ID:   "script-step",
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {}",
		},
	}))
	assert.ErrorIs(t, eng.RegisterStep(&api.Step{
		ID:   "http-step",
		Name: "HTTP Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.test"},
		},
	}), api.ErrInvalidStepType)
	assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
		ID:    "local-flow",
		Goals: []api.StepID{st.ID},
	}))

	select {
	case sid := <-ran:
		assert.Equal(t, st.ID, sid)
	case <-time.After(5 * time.Second):
		t.Fatal("handler was never executed")
	}

	assert.Eventually(t, func() bool {
		fl, err := eng.GetFlowState("local-flow")
		return err == nil && fl.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)
}

func TestConfiguredHandlersAreExact(t *testing.T) {
	eng := newTestEngine(t, argyll.Options{Handlers: step.Handlers{}})
	err := eng.RegisterStep(&api.Step{
		ID:   "script-step",
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {}",
		},
	})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}
