package builtins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestFlowHandlerHasNoExternalExecution(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeFlow)
	assert.NoError(t, err)
	assert.Nil(t, handler.Invoke)
}

func TestFlowChildrenNilFlow(t *testing.T) {
	reg := newRegistry(&testClient{})
	ids, err := reg.Children(&api.Step{Type: api.StepTypeFlow})
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestFlowChildrenGoals(t *testing.T) {
	reg := newRegistry(&testClient{})
	st := &api.Step{
		Type: api.StepTypeFlow,
		Flow: &api.FlowConfig{Goals: api.Goals{Steps: []api.StepID{"a", "b"}}},
	}
	goals, err := reg.Children(st)
	assert.NoError(t, err)
	assert.Same(t, &st.Flow.Goals, goals)
}
