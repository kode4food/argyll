package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestValidateSuccess(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"input1", "input2", "input3"},
	}

	args := api.Args{
		"input1": "value1",
		"input2": "value2",
		"input3": "value3",
	}

	err := plan.ValidateInputs(args)
	assert.NoError(t, err)
}

func TestValidateExtraArgs(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"input1"},
	}

	args := api.Args{
		"input1": "value1",
		"extra1": "extra_value",
		"extra2": "another_value",
	}

	err := plan.ValidateInputs(args)
	assert.NoError(t, err)
}

func TestValidateMissing(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"required_input"},
	}

	args := api.Args{
		"other_input": "value",
	}

	err := plan.ValidateInputs(args)
	require.Error(t, err)

	expected := "required inputs not provided: [required_input]"
	assert.Equal(t, expected, err.Error())
}

func TestValidateMissingMulti(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"input1", "input2", "input3"},
	}

	args := api.Args{
		"input1": "value1",
	}

	err := plan.ValidateInputs(args)
	require.Error(t, err)

	errorMsg := err.Error()
	assert.True(t,
		errorMsg == "required inputs not provided: [input2 input3]" ||
			errorMsg == "required inputs not provided: [input3 input2]")
}

func TestValidateNoRequired(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{},
	}

	args := api.Args{}

	err := plan.ValidateInputs(args)
	assert.NoError(t, err)
}

func TestValidateNilArgs(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"input1"},
	}

	err := plan.ValidateInputs(nil)
	assert.Error(t, err)
}

func TestValidateEmptyArgs(t *testing.T) {
	plan := &api.ExecutionPlan{
		Required: []api.Name{"input1", "input2"},
	}

	args := api.Args{}

	err := plan.ValidateInputs(args)
	assert.Error(t, err)
}

func TestBuildDependencies(t *testing.T) {
	steps := api.Steps{
		"step-a": &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Attributes: api.AttributeSpecs{
				"input1": {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		},
		"step-b": &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleRequired, Type: api.TypeString},
				"final":  {Role: api.RoleOutput, Type: api.TypeString},
			},
		},
		"step-c": &api.Step{
			ID:   "step-c",
			Name: "Step C",
			Attributes: api.AttributeSpecs{
				"input1": {Role: api.RoleRequired, Type: api.TypeString},
				"data":   {Role: api.RoleOutput, Type: api.TypeString},
			},
		},
	}

	deps := api.BuildDependencies(steps)

	assert.Contains(t, deps, api.Name("input1"))
	assert.Contains(t, deps, api.Name("result"))
	assert.Contains(t, deps, api.Name("final"))
	assert.Contains(t, deps, api.Name("data"))

	assert.Len(t, deps["input1"].Providers, 0)
	assert.Len(t, deps["input1"].Consumers, 2)
	assert.Contains(t, deps["input1"].Consumers, api.StepID("step-a"))
	assert.Contains(t, deps["input1"].Consumers, api.StepID("step-c"))

	assert.Len(t, deps["result"].Providers, 1)
	assert.Len(t, deps["result"].Consumers, 1)
	assert.Contains(t, deps["result"].Providers, api.StepID("step-a"))
	assert.Contains(t, deps["result"].Consumers, api.StepID("step-b"))

	assert.Len(t, deps["final"].Providers, 1)
	assert.Len(t, deps["final"].Consumers, 0)
	assert.Contains(t, deps["final"].Providers, api.StepID("step-b"))

	assert.Len(t, deps["data"].Providers, 1)
	assert.Len(t, deps["data"].Consumers, 0)
	assert.Contains(t, deps["data"].Providers, api.StepID("step-c"))
}

func TestMultipleProviders(t *testing.T) {
	steps := api.Steps{
		"step-a": &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Attributes: api.AttributeSpecs{
				"data": {Role: api.RoleOutput, Type: api.TypeString},
			},
		},
		"step-b": &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Attributes: api.AttributeSpecs{
				"data": {Role: api.RoleOutput, Type: api.TypeString},
			},
		},
	}

	deps := api.BuildDependencies(steps)

	assert.Len(t, deps["data"].Providers, 2)
	assert.Contains(t, deps["data"].Providers, api.StepID("step-a"))
	assert.Contains(t, deps["data"].Providers, api.StepID("step-b"))
}

func TestEmptySteps(t *testing.T) {
	steps := api.Steps{}
	deps := api.BuildDependencies(steps)
	assert.Empty(t, deps)
}

func TestNoAttributes(t *testing.T) {
	steps := api.Steps{
		"step-a": &api.Step{
			ID:         "step-a",
			Name:       "Step A",
			Attributes: api.AttributeSpecs{},
		},
	}

	deps := api.BuildDependencies(steps)
	assert.Empty(t, deps)
}
