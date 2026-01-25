package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSet(t *testing.T) {
	original := api.Args{
		"existing": "value",
	}

	result := original.Set("new_key", "new_value")

	assert.Equal(t, "new_value", result["new_key"])
	assert.Equal(t, "value", result["existing"])
	assert.NotContains(t,
		original, "new_key", "Set should not modify original Args",
	)
}

func TestSetOverwriteExisting(t *testing.T) {
	original := api.Args{
		"key": "old_value",
	}

	result := original.Set("key", "new_value")

	assert.Equal(t, "new_value", result["key"])
	assert.Equal(t, "old_value", original["key"],
		"Set should not modify original Args",
	)
}

func TestGetString(t *testing.T) {
	args := api.Args{
		"string_key":  "test_value",
		"int_key":     42,
		"bool_key":    true,
		"missing_key": nil,
	}

	t.Run("existing_string", func(t *testing.T) {
		result := args.GetString("string_key", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("non_existent_key", func(t *testing.T) {
		result := args.GetString("nonexistent", "default_value")
		assert.Equal(t, "default_value", result)
	})

	t.Run("wrong_type", func(t *testing.T) {
		result := args.GetString("int_key", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("nil_value", func(t *testing.T) {
		result := args.GetString("missing_key", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("empty_default", func(t *testing.T) {
		result := args.GetString("nonexistent", "")
		assert.Empty(t, result)
	})
}

func TestGetBool(t *testing.T) {
	args := api.Args{
		"bool_true":   true,
		"bool_false":  false,
		"string_key":  "not_a_bool",
		"int_key":     1,
		"missing_key": nil,
	}

	t.Run("existing_true", func(t *testing.T) {
		result := args.GetBool("bool_true", false)
		assert.True(t, result)
	})

	t.Run("existing_false", func(t *testing.T) {
		result := args.GetBool("bool_false", true)
		assert.False(t, result)
	})

	t.Run("non_existent_key_default_true", func(t *testing.T) {
		result := args.GetBool("nonexistent", true)
		assert.True(t, result)
	})

	t.Run("non_existent_key_default_false", func(t *testing.T) {
		result := args.GetBool("nonexistent", false)
		assert.False(t, result)
	})

	t.Run("wrong_type", func(t *testing.T) {
		result := args.GetBool("string_key", true)
		assert.True(t, result)
	})

	t.Run("nil_value", func(t *testing.T) {
		result := args.GetBool("missing_key", true)
		assert.True(t, result)
	})
}

func TestGetInt(t *testing.T) {
	args := api.Args{
		"int_key":     42,
		"float_key":   3.14,
		"string_key":  "not_an_int",
		"bool_key":    true,
		"missing_key": nil,
	}

	t.Run("existing_int", func(t *testing.T) {
		result := args.GetInt("int_key", 0)
		assert.Equal(t, 42, result)
	})

	t.Run("float_to_int", func(t *testing.T) {
		result := args.GetInt("float_key", 0)
		assert.Equal(t, 3, result)
	})

	t.Run("non_existent_key", func(t *testing.T) {
		result := args.GetInt("nonexistent", 999)
		assert.Equal(t, 999, result)
	})

	t.Run("wrong_type", func(t *testing.T) {
		result := args.GetInt("string_key", 100)
		assert.Equal(t, 100, result)
	})

	t.Run("nil_value", func(t *testing.T) {
		result := args.GetInt("missing_key", 50)
		assert.Equal(t, 50, result)
	})

	t.Run("negative_int", func(t *testing.T) {
		testArgs := api.Args{"negative": -10}
		result := testArgs.GetInt("negative", 0)
		assert.Equal(t, -10, result)
	})

	t.Run("zero_default", func(t *testing.T) {
		result := args.GetInt("nonexistent", 0)
		assert.Equal(t, 0, result)
	})
}

func TestGetIntFloatConversion(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected int
	}{
		{name: "positive_float", value: 10.7, expected: 10},
		{name: "negative_float", value: -5.9, expected: -5},
		{name: "zero_float", value: 0.0, expected: 0},
		{name: "large_float", value: 999999.99, expected: 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := api.Args{"value": tt.value}
			result := args.GetInt("value", -1)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChaining(t *testing.T) {
	original := api.Args{}

	result := original.
		Set("key1", "value1").
		Set("key2", 42).
		Set("key3", true)

	assert.Len(t, result, 3)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, 42, result["key2"])
	assert.Equal(t, true, result["key3"])
	assert.Empty(t, original)
}
