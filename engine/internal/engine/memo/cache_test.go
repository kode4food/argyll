package memo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/memo"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCacheGetPut(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheMiss(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheDifferentInputs(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheInvalidatesOnStepChange(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheKeepsOnMetadataChange(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheEmptyInputs(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheHashAttributeOrder(t *testing.T) {
	cache := memo.NewCache(100)

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

func TestCacheHashFlowConfig(t *testing.T) {
	cache := memo.NewCache(100)

	flowConfig1 := &api.FlowConfig{
		Goals: []api.StepID{"goal1"},
	}
	flowConfig2 := &api.FlowConfig{
		Goals: []api.StepID{"goal1"},
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

func TestCacheHashInputOrder(t *testing.T) {
	cache := memo.NewCache(100)

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
