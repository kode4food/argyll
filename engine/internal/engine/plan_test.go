package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestNoGoals(t *testing.T) {
	eng := &engine.Engine{}
	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{},
	}

	_, err := eng.CreateExecutionPlan(engState, []api.StepID{}, api.Args{})
	assert.Error(t, err)
}

func TestGoalStepNotFound(t *testing.T) {
	eng := &engine.Engine{}
	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{},
	}

	_, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"nonexistent"}, api.Args{},
	)
	assert.Error(t, err)
}

func TestSimpleResolver(t *testing.T) {
	eng := &engine.Engine{}

	resolverStep := &api.Step{
		ID:      "resolver",
		Name:    "Resolver",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"resolver": resolverStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"resolver"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Goals, 1)
	assert.Equal(t, api.StepID("resolver"), plan.Goals[0])

	assert.Len(t, plan.Steps, 1)
	assert.Contains(t, plan.Steps, api.StepID("resolver"))

	assert.Empty(t, plan.Required)
}

func TestProcessorWithInit(t *testing.T) {
	eng := &engine.Engine{}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"processor": processorStep,
		},
	}

	initState := api.Args{"input": "test-value"}
	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor"}, initState,
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 1)
	assert.Empty(t, plan.Required)
}

func TestProcessorNoInit(t *testing.T) {
	eng := &engine.Engine{}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"processor": processorStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Required, 1)
	assert.Equal(t, api.Name("input"), plan.Required[0])
}

func TestChained(t *testing.T) {
	eng := &engine.Engine{}

	resolverStep := &api.Step{
		ID:      "resolver",
		Name:    "Resolver",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"data":   {Role: api.RoleRequired, Type: api.TypeString},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	collectorStep := &api.Step{
		ID:      "collector",
		Name:    "Collector",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"result": {Role: api.RoleRequired, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"resolver":  resolverStep,
			"processor": processorStep,
			"collector": collectorStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"collector"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 3)

	assert.Contains(t, plan.Steps, api.StepID("resolver"))
	assert.Contains(t, plan.Steps, api.StepID("processor"))
	assert.Contains(t, plan.Steps, api.StepID("collector"))

	assert.Empty(t, plan.Required)
}

func TestMultipleGoals(t *testing.T) {
	eng := &engine.Engine{}

	step1 := &api.Step{
		ID:      "step1",
		Name:    "Step 1",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"output1": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	step2 := &api.Step{
		ID:      "step2",
		Name:    "Step 2",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"output2": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"step1": step1,
			"step2": step2,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"step1", "step2"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Goals, 2)
	assert.Len(t, plan.Steps, 2)
}

func TestExistingOutputs(t *testing.T) {
	eng := &engine.Engine{}

	step := &api.Step{
		ID:      "step",
		Name:    "Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"step": step,
		},
	}

	initState := api.Args{"data": "already-available"}
	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"step"}, initState,
	)
	require.NoError(t, err)

	assert.Empty(t, plan.Steps)
}

func TestComplexGraph(t *testing.T) {
	eng := &engine.Engine{}

	resolver1 := &api.Step{
		ID:      "resolver1",
		Name:    "Resolver 1",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"a": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	resolver2 := &api.Step{
		ID:      "resolver2",
		Name:    "Resolver 2",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"b": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processor1 := &api.Step{
		ID:      "processor1",
		Name:    "Processor 1",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"a": {Role: api.RoleRequired, Type: api.TypeString},
			"b": {Role: api.RoleRequired, Type: api.TypeString},
			"c": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processor2 := &api.Step{
		ID:      "processor2",
		Name:    "Processor 2",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"c": {Role: api.RoleRequired, Type: api.TypeString},
			"d": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"resolver1":  resolver1,
			"resolver2":  resolver2,
			"processor1": processor1,
			"processor2": processor2,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor2"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 4)

	requiredSteps := []api.StepID{
		"resolver1", "resolver2", "processor1", "processor2",
	}
	for _, stepID := range requiredSteps {
		assert.Contains(t, plan.Steps, stepID)
	}

	assert.Empty(t, plan.Required)
}

func TestReceipts(t *testing.T) {
	eng := &engine.Engine{}

	step := &api.Step{
		ID:      "step",
		Name:    "Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"step": step,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"step"}, api.Args{},
	)
	require.NoError(t, err)

	// Verify plan was created successfully
	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 1)
}

func TestMissingDependency(t *testing.T) {
	eng := &engine.Engine{}

	step := &api.Step{
		ID:      "step",
		Name:    "Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"nonexistent": {Role: api.RoleRequired, Type: api.TypeString},
			"output":      {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"step": step,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"step"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Required, 1)
	assert.Equal(t, api.Name("nonexistent"), plan.Required[0])
}

func TestOptionalInput(t *testing.T) {
	eng := &engine.Engine{}

	resolverStep := &api.Step{
		ID:      "resolver",
		Name:    "Resolver",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"optional_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"resolver":  resolverStep,
			"processor": processorStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 2)

	assert.Contains(t, plan.Steps, api.StepID("resolver"))
	assert.Contains(t, plan.Steps, api.StepID("processor"))

	assert.Empty(t, plan.Required)
}

func TestOptionalMissing(t *testing.T) {
	eng := &engine.Engine{}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"processor": processorStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 1)
	assert.Contains(t, plan.Steps, api.StepID("processor"))

	assert.Empty(t, plan.Required)
}

func TestMixedInputs(t *testing.T) {
	eng := &engine.Engine{}

	resolver1 := &api.Step{
		ID:      "resolver1",
		Name:    "Resolver 1",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"required_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	resolver2 := &api.Step{
		ID:      "resolver2",
		Name:    "Resolver 2",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"optional_data": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	processorStep := &api.Step{
		ID:      "processor",
		Name:    "Processor",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: map[api.Name]*api.AttributeSpec{
			"required_data": {Role: api.RoleRequired, Type: api.TypeString},
			"optional_data": {Role: api.RoleOptional, Type: api.TypeString},
			"result":        {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
			Timeout:  30 * api.Second,
		},
	}

	engState := &api.EngineState{
		Steps: map[api.StepID]*api.Step{
			"resolver1": resolver1,
			"resolver2": resolver2,
			"processor": processorStep,
		},
	}

	plan, err := eng.CreateExecutionPlan(
		engState, []api.StepID{"processor"}, api.Args{},
	)
	require.NoError(t, err)

	assert.Len(t, plan.Steps, 3)

	assert.Contains(t, plan.Steps, api.StepID("resolver1"))
	assert.Contains(t, plan.Steps, api.StepID("resolver2"))
	assert.Contains(t, plan.Steps, api.StepID("processor"))

	assert.Empty(t, plan.Required)
}
