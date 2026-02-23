package api_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCreateFlowRequestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: []api.StepID{"step-1"},
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty id", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			Goals: []api.StepID{"step-1"},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDEmpty)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    "my:flow",
			Goals: []api.StepID{"step-1"},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDInvalid)
	})

	t.Run("no goals", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID: "my-flow",
		}
		assert.ErrorIs(t, req.Validate(), api.ErrGoalsRequired)
	})

	t.Run("ID too long", func(t *testing.T) {
		req := &api.CreateFlowRequest{
			ID:    api.FlowID(strings.Repeat("a", api.MaxFlowIDLen+1)),
			Goals: []api.StepID{"step-1"},
		}
		assert.ErrorIs(t, req.Validate(), api.ErrFlowIDTooLong)
	})

	t.Run("too many goals", func(t *testing.T) {
		goals := make([]api.StepID, api.MaxGoalCount+1)
		for i := range goals {
			goals[i] = api.StepID(fmt.Sprintf("step-%d", i))
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: goals,
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyGoals)
	})

	t.Run("too many init keys", func(t *testing.T) {
		init := api.Args{}
		for i := range api.MaxInitKeys + 1 {
			init[api.Name(fmt.Sprintf("key-%d", i))] = "value"
		}
		req := &api.CreateFlowRequest{
			ID:    "my-flow",
			Goals: []api.StepID{"step-1"},
			Init:  init,
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyInit)
	})

	t.Run("too many labels", func(t *testing.T) {
		labels := api.Labels{}
		for i := range api.MaxLabelCount + 1 {
			labels[fmt.Sprintf("label-%d", i)] = "value"
		}
		req := &api.CreateFlowRequest{
			ID:     "my-flow",
			Goals:  []api.StepID{"step-1"},
			Labels: labels,
		}
		assert.ErrorIs(t, req.Validate(), api.ErrTooManyLabels)
	})
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{name: "clean id", input: "my-flow", expected: "my-flow"},
		{name: "uppercase lowercased", input: "My-Flow", expected: "my-flow"},
		{name: "spaces become hyphens", input: "my flow", expected: "my-flow"},
		{name: "colons stripped", input: "my:flow", expected: "myflow"},
		{
			name: "leading hyphen trimmed", input: "-my-flow",
			expected: "my-flow",
		},
		{
			name: "trailing hyphen trimmed", input: "my-flow-",
			expected: "my-flow",
		},
		{name: "invalid chars stripped", input: "my@flow!", expected: "myflow"},
		{name: "empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t,
				api.FlowID(tt.expected),
				api.SanitizeID(api.FlowID(tt.input)),
			)
			assert.Equal(t,
				api.StepID(tt.expected),
				api.SanitizeID(api.StepID(tt.input)),
			)
		})
	}
}
