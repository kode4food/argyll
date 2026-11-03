package util

import (
	"maps"
	"slices"
)

// EqualJSON compares JSON values decoded into any, plus int. An int matches an
// equal float64, and any other type is never equal
func EqualJSON(a, b any) bool {
	switch a := a.(type) {
	case map[string]any:
		b, ok := b.(map[string]any)
		return ok && maps.EqualFunc(a, b, EqualJSON)
	case []any:
		b, ok := b.([]any)
		return ok && slices.EqualFunc(a, b, EqualJSON)
	case float64:
		if b, ok := b.(int); ok {
			return intEqualsFloat(b, a)
		}
		return a == b
	case int:
		if b, ok := b.(float64); ok {
			return intEqualsFloat(a, b)
		}
		return a == b
	case nil, bool, string:
		return a == b
	default:
		return false
	}
}

// intEqualsFloat is exact: the round trip rejects floats that only match after
// the int rounds
func intEqualsFloat(i int, f float64) bool {
	return f < 1<<63 && float64(i) == f && int(f) == i
}
