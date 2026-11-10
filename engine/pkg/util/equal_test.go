package util_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/util"
)

func TestEqualJSON(t *testing.T) {
	for _, tt := range []struct {
		name  string
		left  any
		right any
		equal bool
	}{
		{
			name:  "nested values",
			left:  map[string]any{"a": []any{true, 4.0}, "b": nil},
			right: map[string]any{"b": nil, "a": []any{true, 4.0}},
			equal: true,
		},
		{
			name:  "missing key",
			left:  map[string]any{"a": nil},
			right: map[string]any{"b": nil},
		},
		{
			name:  "array order",
			left:  []any{1.0, 2.0},
			right: []any{2.0, 1.0},
		},
		{
			name:  "mixed types",
			left:  "1",
			right: []any{"1"},
		},
		{
			name:  "numbers",
			left:  1.5,
			right: 1.5,
			equal: true,
		},
		{
			name:  "different numbers",
			left:  1.0,
			right: 1.5,
		},
		{
			name:  "NaN",
			left:  math.NaN(),
			right: math.NaN(),
		},
		{
			name:  "number and string",
			left:  7.0,
			right: "7",
		},
		{
			name:  "null and false",
			left:  nil,
			right: false,
		},
		{
			name:  "empty array",
			left:  []any{},
			right: nil,
		},
		{
			name:  "empty object",
			left:  map[string]any{},
			right: nil,
		},
		{
			name:  "non-JSON number",
			left:  json.Number("7"),
			right: 7.0,
		},
		{
			name:  "ints",
			left:  7,
			right: 7,
			equal: true,
		},
		{
			name:  "different ints",
			left:  7,
			right: 8,
		},
		{
			name:  "int and float",
			left:  7,
			right: 7.0,
			equal: true,
		},
		{
			name:  "int and fraction",
			left:  7,
			right: 7.5,
		},
		{
			name:  "int and string",
			left:  7,
			right: "7",
		},
		{
			name:  "large int",
			left:  1<<53 + 1,
			right: float64(1 << 53),
		},
		{
			name:  "int overflow",
			left:  1<<63 - 1,
			right: float64(1 << 63),
		},
		{
			name:  "nested ints",
			left:  map[string]any{"count": 1},
			right: map[string]any{"count": 1.0},
			equal: true,
		},
		{
			name:  "non-JSON values",
			left:  int64(7),
			right: int64(7),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, util.EqualJSON(tt.left, tt.right))
			assert.Equal(t, tt.equal, util.EqualJSON(tt.right, tt.left))
		})
	}
}
