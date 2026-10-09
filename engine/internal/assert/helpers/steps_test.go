package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStep(t *testing.T) {
	st := helpers.NewTestStep()

	assert.NotNil(t, st)
	assert.NotEmpty(t, st.ID)
	assert.Equal(t, api.Name("Test Step"), st.Name)
	assert.Equal(t, api.StepTypeService, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.NotEmpty(t, st.HTTP.Invoke.Endpoint)

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepWithArgs(t *testing.T) {
	req := []api.Name{"req1", "req2"}
	opt := []api.Name{"opt1", "opt2"}

	st := helpers.NewTestStepWithArgs(req, opt)

	assert.NotNil(t, st)
	requiredArgs := st.GetRequiredArgs()
	optionalArgs := st.GetOptionalArgs()
	assert.Len(t, requiredArgs, 2)
	assert.Len(t, optionalArgs, 2)

	assert.Contains(t, requiredArgs, api.Name("req1"))
	assert.Contains(t, requiredArgs, api.Name("req2"))
	assert.Contains(t, optionalArgs, api.Name("opt1"))
	assert.Contains(t, optionalArgs, api.Name("opt2"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestSimpleStep(t *testing.T) {
	st := helpers.NewSimpleStep("test-id")

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("test-id"), st.ID)
	assert.Equal(t, api.StepTypeService, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.Empty(t, st.GetRequiredArgs())
	assert.Empty(t, st.GetOptionalArgs())
	assert.Empty(t, st.GetOutputArgs())

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepWithOutput(t *testing.T) {
	st := helpers.NewStepWithOutputs("output-step", "result1", "result2")

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("output-step"), st.ID)
	outputArgs := st.GetOutputArgs()
	assert.Len(t, outputArgs, 2)
	assert.Contains(t, outputArgs, api.Name("result1"))
	assert.Contains(t, outputArgs, api.Name("result2"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestScriptStep(t *testing.T) {
	st := helpers.NewScriptStep(
		"script-id", api.ScriptLangLua, "return {result = 42}", "result",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("script-id"), st.ID)
	assert.Equal(t, api.StepTypeScript, st.Type)
	assert.NotNil(t, st.Script)
	assert.Equal(t, api.ScriptLangLua, st.Script.Language)
	assert.Equal(t, "return {result = 42}", st.Script.Script)
	assert.Len(t, st.GetOutputArgs(), 1)
	assert.Contains(t, st.GetOutputArgs(), api.Name("result"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestScriptNoOutput(t *testing.T) {
	st := helpers.NewScriptStep(
		"script-id", api.ScriptLangLua, "return {}",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepTypeScript, st.Type)
	assert.Empty(t, st.GetOutputArgs())
}

func TestStepPredicate(t *testing.T) {
	st := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangLua, "return true", "output",
	)

	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("pred-step"), st.ID)
	assert.Equal(t, api.StepTypeService, st.Type)
	assert.NotNil(t, st.HTTP)
	assert.NotNil(t, st.Predicate)
	assert.Equal(t, api.ScriptLangLua, st.Predicate.Language)
	assert.Equal(t, "return true", st.Predicate.Script)
	assert.Len(t, st.GetOutputArgs(), 1)
	assert.Contains(t, st.GetOutputArgs(), api.Name("output"))

	err := st.Validate()
	assert.NoError(t, err)
}

func TestStepPredicateNoOutput(t *testing.T) {
	st := helpers.NewStepWithPredicate(
		"pred-step", api.ScriptLangLua, "return false",
	)

	assert.NotNil(t, st)
	assert.NotNil(t, st.Predicate)
	assert.Empty(t, st.GetOutputArgs())
}
