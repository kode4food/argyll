package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestGetMetaString(t *testing.T) {
	t.Run("returns value when key exists", func(t *testing.T) {
		meta := api.Metadata{"flow_id": "test-flow"}
		val, ok := api.GetMetaString[api.FlowID](meta, "flow_id")
		assert.True(t, ok)
		assert.Equal(t, api.FlowID("test-flow"), val)
	})

	t.Run("returns false when key missing", func(t *testing.T) {
		meta := api.Metadata{}
		val, ok := api.GetMetaString[api.FlowID](meta, "flow_id")
		assert.False(t, ok)
		assert.Equal(t, api.FlowID(""), val)
	})

	t.Run("returns false when value is empty string", func(t *testing.T) {
		meta := api.Metadata{"flow_id": ""}
		val, ok := api.GetMetaString[api.FlowID](meta, "flow_id")
		assert.False(t, ok)
		assert.Equal(t, api.FlowID(""), val)
	})

	t.Run("converts string to typed string", func(t *testing.T) {
		meta := api.Metadata{"step_id": "test-step"}
		val, ok := api.GetMetaString[api.StepID](meta, "step_id")
		assert.True(t, ok)
		assert.Equal(t, api.StepID("test-step"), val)
	})

	t.Run("returns false for non-string value", func(t *testing.T) {
		meta := api.Metadata{"count": 42}
		val, ok := api.GetMetaString[api.FlowID](meta, "count")
		assert.False(t, ok)
		assert.Equal(t, api.FlowID(""), val)
	})

	t.Run("handles typed string values", func(t *testing.T) {
		meta := api.Metadata{"parent": api.FlowID("parent-flow")}
		val, ok := api.GetMetaString[api.FlowID](meta, "parent")
		assert.True(t, ok)
		assert.Equal(t, api.FlowID("parent-flow"), val)
	})
}
