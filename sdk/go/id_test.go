package argyll_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	argyll "github.com/kode4food/argyll/sdk/go"
)

func TestNewFlowID(t *testing.T) {
	id := argyll.NewFlowID("test-flow")

	assert.True(t, strings.HasPrefix(string(id), "test-flow-"))

	parts := strings.Split(string(id), "-")
	assert.GreaterOrEqual(t, len(parts), 3)

	suffix := parts[len(parts)-1]
	assert.Equal(t, 6, len(suffix))
	assert.Regexp(t, "^[0-9a-f]{6}$", suffix)
}

func TestNewFlowIDUniqueness(t *testing.T) {
	id1 := argyll.NewFlowID("test")
	id2 := argyll.NewFlowID("test")

	assert.NotEqual(t, id1, id2)
}

func TestNewFlowIDSanitization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple", "simple-"},
		{"With Spaces", "with-spaces-"},
		{"UPPERCASE", "uppercase-"},
		{"Mixed-Case_Test", "mixed-case_test-"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id := argyll.NewFlowID(tt.input)
			assert.True(t, strings.HasPrefix(string(id), tt.expected))
		})
	}
}
