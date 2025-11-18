package builder_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/builder"
)

func TestNewFlowID(t *testing.T) {
	id := builder.NewFlowID("test-flow")

	assert.True(t, strings.HasPrefix(string(id), "test-flow-"))

	parts := strings.Split(string(id), "-")
	assert.GreaterOrEqual(t, len(parts), 3)

	suffix := parts[len(parts)-1]
	assert.Equal(t, 6, len(suffix))
	assert.Regexp(t, "^[0-9a-f]{6}$", suffix)
}

func TestNewFlowIDUniqueness(t *testing.T) {
	id1 := builder.NewFlowID("test")
	id2 := builder.NewFlowID("test")

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
			id := builder.NewFlowID(tt.input)
			assert.True(t, strings.HasPrefix(string(id), tt.expected))
		})
	}
}
