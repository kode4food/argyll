package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

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
			name:     "leading hyphen trimmed",
			input:    "-my-flow",
			expected: "my-flow",
		},
		{
			name:     "trailing hyphen trimmed",
			input:    "my-flow-",
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
