package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMemoCache_GetPut(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"input":  &api.AttributeSpec{Role: api.RoleRequired},
			"output": &api.AttributeSpec{Role: api.RoleOutput},
		},
	}

	inputs := api.Args{"input": "value"}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step, inputs, outputs)
	assert.NoError(t, err)

	result, ok := cache.Get(step, inputs)
	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}

func TestMemoCache_CacheMiss(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs := api.Args{"input": "value"}

	_, ok := cache.Get(step, inputs)
	assert.False(t, ok)
}

func TestMemoCache_DifferentInputsDifferentCache(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs1 := api.Args{"input": "value1"}
	outputs1 := api.Args{"output": "result1"}

	inputs2 := api.Args{"input": "value2"}
	outputs2 := api.Args{"output": "result2"}

	err := cache.Put(step, inputs1, outputs1)
	assert.NoError(t, err)
	err = cache.Put(step, inputs2, outputs2)
	assert.NoError(t, err)

	result, ok := cache.Get(step, inputs1)
	assert.True(t, ok)
	assert.Equal(t, outputs1, result)

	result, ok = cache.Get(step, inputs2)
	assert.True(t, ok)
	assert.Equal(t, outputs2, result)
}

func TestMemoCache_StepDefinitionChangeInvalidatesCache(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step1 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs := api.Args{"input": "value"}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step1, inputs, outputs)
	assert.NoError(t, err)

	step2 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://different.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	_, ok := cache.Get(step2, inputs)
	assert.False(t, ok)
}

func TestMemoCache_StepMetadataChangeDoesNotInvalidateCache(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step1 := &api.Step{
		ID:   api.StepID("test"),
		Name: "Original Name",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Labels: api.Labels{"env": "test"},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs := api.Args{"input": "value"}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step1, inputs, outputs)
	assert.NoError(t, err)

	step2 := &api.Step{
		ID:   api.StepID("test"),
		Name: "Different Name",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Labels: api.Labels{"env": "prod"},
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	result, ok := cache.Get(step2, inputs)
	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}

func TestMemoCache_EmptyInputs(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"output": &api.AttributeSpec{Role: api.RoleOutput},
		},
	}

	inputs := api.Args{}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step, inputs, outputs)
	assert.NoError(t, err)

	result, ok := cache.Get(step, inputs)
	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}

func TestMemoCache_NilStep(t *testing.T) {
	cache := engine.NewMemoCache(100)

	inputs := api.Args{"input": "value"}

	_, ok := cache.Get(nil, inputs)
	assert.False(t, ok)

	err := cache.Put(nil, inputs, api.Args{"output": "result"})
	assert.NoError(t, err)
}

func TestMemoCache_NilCache(t *testing.T) {
	var cache *engine.MemoCache

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
	}

	inputs := api.Args{"input": "value"}

	_, ok := cache.Get(step, inputs)
	assert.False(t, ok)

	err := cache.Put(step, inputs, api.Args{"output": "result"})
	assert.NoError(t, err)
}

func TestMemoCache_DeterministicHashingWithAttributeOrder(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step1 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"z": &api.AttributeSpec{Role: api.RoleRequired},
			"a": &api.AttributeSpec{Role: api.RoleRequired},
			"m": &api.AttributeSpec{Role: api.RoleOutput},
		},
	}

	step2 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"a": &api.AttributeSpec{Role: api.RoleRequired},
			"m": &api.AttributeSpec{Role: api.RoleOutput},
			"z": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs := api.Args{"a": "1", "z": "2", "m": "3"}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step1, inputs, outputs)
	assert.NoError(t, err)
	result, ok := cache.Get(step2, inputs)

	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}

func TestMemoCache_DeterministicHashingWithFlowConfig(t *testing.T) {
	cache := engine.NewMemoCache(100)

	flowConfig1 := &api.FlowConfig{
		Goals: []api.StepID{"goal1"},
		InputMap: map[api.Name]api.Name{
			"z": "z_mapped",
			"a": "a_mapped",
		},
		OutputMap: map[api.Name]api.Name{
			"m": "m_mapped",
			"b": "b_mapped",
		},
	}

	flowConfig2 := &api.FlowConfig{
		Goals: []api.StepID{"goal1"},
		InputMap: map[api.Name]api.Name{
			"a": "a_mapped",
			"z": "z_mapped",
		},
		OutputMap: map[api.Name]api.Name{
			"b": "b_mapped",
			"m": "m_mapped",
		},
	}

	step1 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeFlow,
		Flow: flowConfig1,
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	step2 := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeFlow,
		Flow: flowConfig2,
		Attributes: api.AttributeSpecs{
			"input": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs := api.Args{"input": "value"}
	outputs := api.Args{"output": "result"}

	err := cache.Put(step1, inputs, outputs)
	assert.NoError(t, err)
	result, ok := cache.Get(step2, inputs)

	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}

func TestMemoCache_DeterministicHashingWithInputArgOrder(t *testing.T) {
	cache := engine.NewMemoCache(100)

	step := &api.Step{
		ID:   api.StepID("test"),
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30000,
		},
		Attributes: api.AttributeSpecs{
			"x": &api.AttributeSpec{Role: api.RoleRequired},
			"y": &api.AttributeSpec{Role: api.RoleRequired},
		},
	}

	inputs1 := api.Args{
		"x": "value-x",
		"y": "value-y",
	}

	inputs2 := api.Args{
		"y": "value-y",
		"x": "value-x",
	}

	outputs := api.Args{"result": "data"}

	err := cache.Put(step, inputs1, outputs)
	assert.NoError(t, err)
	result, ok := cache.Get(step, inputs2)

	assert.True(t, ok)
	assert.Equal(t, outputs, result)
}
