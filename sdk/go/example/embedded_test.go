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
	eng, err := argyll.New(argyll.Options{
		Backend: func(pub timebox.Publisher) (timebox.Backend, error) {
			return memory.Open(memory.Config{Publisher: pub}), nil
		},
		Handlers: example.ArgyllEmbeddedHandlers(),
	})
	assert.NoError(t, err)
	t.Cleanup(func() { _ = eng.Stop() })
	assert.NoError(t, eng.Start())

	steps, err := example.ArgyllEmbeddedSteps()
	assert.NoError(t, err)
	for _, st := range steps {
		if st.ID == "greet" {
			assert.NoError(t, eng.RegisterStep(st))
		}
	}

	assert.NoError(t, eng.StartFlow(api.CreateFlowRequest{
		ID:    "greet-flow",
		Goals: []api.StepID{"greet"},
		Init:  api.InitArgs{"name": {"argyll"}},
	}))
	assert.Eventually(t, func() bool {
		fl, err := eng.GetFlowState("greet-flow")
		return err == nil && fl.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)

	fl, err := eng.GetFlowState("greet-flow")
	assert.NoError(t, err)
	assert.Equal(t, "hello argyll", fl.GetAttributes()["greeting"])
}
