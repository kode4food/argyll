package example_test

import (
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/argyll"
	"github.com/kode4food/argyll/sdk/go/example"
)

func TestEmbeddedGreet(t *testing.T) {
	eng := embeddedEngine(t)
	for _, st := range example.ArgyllEmbeddedSteps() {
		if st.ID == "greet" {
			assert.Equal(t, api.StepType("greeter"), st.Type)
			assert.NoError(t, eng.RegisterStep(st))
		}
	}

	fl := runFlow(t, eng, api.CreateFlowRequest{
		ID:    "greet-flow",
		Goals: []api.StepID{"greet"},
		Init:  api.InitArgs{"name": {"argyll"}},
	})
	assert.Equal(t, "hello argyll", fl.GetAttributes()["greeting"])
}

func TestEmbeddedSharedType(t *testing.T) {
	eng := embeddedEngine(t)

	// a second Step of the generated type, mapping its own Attributes onto the
	// function's invocation names
	assert.NoError(t, eng.RegisterStep(&api.Step{
		ID:   "welcome",
		Name: "Welcome",
		Type: "greeter",
		Attributes: api.AttributeSpecs{
			"guest": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{Name: "name"},
				},
			},
			"welcome": {
				Role: api.RoleOutput,
				Type: api.TypeString,
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{Name: "greeting"},
				},
			},
		},
	}))

	fl := runFlow(t, eng, api.CreateFlowRequest{
		ID:    "welcome-flow",
		Goals: []api.StepID{"welcome"},
		Init:  api.InitArgs{"guest": {"ada"}},
	})
	assert.Equal(t, "hello ada", fl.GetAttributes()["welcome"])
}

func embeddedEngine(t *testing.T) argyll.Engine {
	t.Helper()
	eng, err := argyll.New(argyll.Options{
		Backend: func(pub timebox.Publisher) (timebox.Backend, error) {
			return memory.Open(memory.Config{Publisher: pub}), nil
		},
		Handlers: example.ArgyllEmbeddedHandlers(),
	})
	assert.NoError(t, err)
	t.Cleanup(func() { _ = eng.Stop() })
	assert.NoError(t, eng.Start())
	return eng
}

func runFlow(
	t *testing.T, eng argyll.Engine, req api.CreateFlowRequest,
) api.FlowState {
	t.Helper()
	assert.NoError(t, eng.StartFlow(req))
	assert.Eventually(t, func() bool {
		fl, err := eng.GetFlowState(req.ID)
		return err == nil && fl.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)

	fl, err := eng.GetFlowState(req.ID)
	assert.NoError(t, err)
	return fl
}
