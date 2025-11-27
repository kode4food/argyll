package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestGetStep(t *testing.T) {
	step1 := &api.Step{ID: "step1", Name: "Step 1"}
	step2 := &api.Step{ID: "step2", Name: "Step 2"}
	step3 := &api.Step{ID: "step3", Name: "Step 3"}

	plan := &api.ExecutionPlan{
		Steps: map[api.StepID]*api.StepInfo{
			"step1": {Step: step1},
			"step2": {Step: step2},
			"step3": {Step: step3},
		},
	}

	t.Run("existing_step", func(t *testing.T) {
		result := plan.GetStep("step2")
		require.NotNil(t, result)
		assert.EqualValues(t, "step2", result.ID)
		assert.EqualValues(t, "Step 2", result.Name)
	})

	t.Run("non_existent_step", func(t *testing.T) {
		result := plan.GetStep("nonexistent")
		assert.Nil(t, result)
	})

	t.Run("first_step", func(t *testing.T) {
		result := plan.GetStep("step1")
		require.NotNil(t, result)
		assert.EqualValues(t, "step1", result.ID)
	})

	t.Run("last_step", func(t *testing.T) {
		result := plan.GetStep("step3")
		require.NotNil(t, result)
		assert.EqualValues(t, "step3", result.ID)
	})
}

func TestGetStepEmptyPlan(t *testing.T) {
	plan := &api.ExecutionPlan{
		Steps: map[api.StepID]*api.StepInfo{},
	}

	result := plan.GetStep("any")
	assert.Nil(t, result)
}

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
