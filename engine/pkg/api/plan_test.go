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

	expected := "required input not provided: 'required_input'"
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
