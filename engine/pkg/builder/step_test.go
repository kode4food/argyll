package builder_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/builder"
)

func TestNewStep(t *testing.T) {
	name := api.Name("Test Step")
	client := testClient()

	step, err := client.NewStep(name).
		WithEndpoint("http://example.com").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepID("test-step"), step.ID)
	assert.Equal(t, name, step.Name)
	assert.Equal(t, "1.0.0", step.Version)
	assert.Equal(t, api.StepTypeSync, step.Type)
	assert.Equal(t, int64(30000), step.HTTP.Timeout)
}

func TestNewStepIDGeneration(t *testing.T) {
	tests := []struct {
		name       string
		stepName   api.Name
		expectedID api.StepID
	}{
		{
			name:       "simple name",
			stepName:   "Test",
			expectedID: "test",
		},
		{
			name:       "multiple words",
			stepName:   "Test Step",
			expectedID: "test-step",
		},
		{
			name:       "camel case",
			stepName:   "TestStepName",
			expectedID: "test-step-name",
		},
		{
			name:       "with underscores",
			stepName:   "test_step_name",
			expectedID: "test-step-name",
		},
		{
			name:       "already snake case",
			stepName:   "test-step",
			expectedID: "test-step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := testClient().NewStep(tt.stepName).
				WithEndpoint("http://example.com").
				Build()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedID, step.ID)
		})
	}
}

func TestWithID(t *testing.T) {
	customID := "custom-id"
	step, err := testClient().NewStep("Test Step").
		WithID(customID).
		WithEndpoint("http://example.com").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepID(customID), step.ID)
	assert.Equal(t, api.Name("Test Step"), step.Name)
}

func TestRequiredArg(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		Required("input1", api.TypeString).
		Required("input2", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.Len(t, step.Attributes, 2)
	assert.Contains(t, step.Attributes, api.Name("input1"))
	assert.EqualValues(t, api.TypeString, step.Attributes["input1"].Type)
	assert.EqualValues(t, api.RoleRequired, step.Attributes["input1"].Role)
}

func TestOptionalArg(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		Optional("optional1", api.TypeString, "").
		Optional("optional2", api.TypeNumber, "42").
		Build()

	assert.NoError(t, err)
	assert.Len(t, step.Attributes, 2)
	assert.Contains(t, step.Attributes, api.Name("optional1"))
	assert.EqualValues(t, api.RoleOptional, step.Attributes["optional1"].Role)
	assert.EqualValues(t, "42", step.Attributes["optional2"].Default)
}

func TestOutputArg(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		Output("output1", api.TypeString).
		Output("output2", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.Len(t, step.Attributes, 2)
	assert.Contains(t, step.Attributes, api.Name("output1"))
	assert.EqualValues(t, api.TypeString, step.Attributes["output1"].Type)
	assert.EqualValues(t, api.RoleOutput, step.Attributes["output1"].Role)
}

func TestWithVersion(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithVersion("2.0.0").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", step.Version)
}

func TestWithEndpoint(t *testing.T) {
	endpoint := "http://example.com/step"
	step, err := testClient().NewStep("Test").
		WithEndpoint(endpoint).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.HTTP)
	assert.Equal(t, endpoint, step.HTTP.Endpoint)
	assert.Equal(t, api.StepTypeSync, step.Type)
}

func TestWithScript(t *testing.T) {
	script := "{:result (+ 1 2)}"
	step, err := testClient().NewStep("Test").
		WithScript(script).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.Script)
	assert.Equal(t, script, step.Script.Script)
	assert.Equal(t, api.ScriptLangAle, step.Script.Language)
	assert.Equal(t, api.StepTypeScript, step.Type)
}

func TestWithScriptLanguage(t *testing.T) {
	script := "custom script"
	lang := "custom-lang"
	step, err := testClient().NewStep("Test").
		WithScriptLanguage(lang, script).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.Script)
	assert.Equal(t, script, step.Script.Script)
	assert.Equal(t, lang, step.Script.Language)
	assert.Equal(t, api.StepTypeScript, step.Type)
}

func TestWithHealthCheck(t *testing.T) {
	healthCheck := "http://example.com/health"
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com/step").
		WithHealthCheck(healthCheck).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.HTTP)
	assert.Equal(t, healthCheck, step.HTTP.HealthCheck)
}

func TestWithTimeout(t *testing.T) {
	timeout := api.Minute
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithTimeout(timeout).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, timeout, step.HTTP.Timeout)
}

func TestWithType(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithType(api.StepTypeAsync).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, step.Type)
}

func TestWithAsyncExecution(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithAsyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, step.Type)
}

func TestWithSyncExecution(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithSyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeSync, step.Type)
}

func TestWithScriptExecution(t *testing.T) {
	step, err := testClient().NewStep("Test").
		WithScript("{:result 42}").
		WithScriptExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeScript, step.Type)
}

func TestWithPredicate(t *testing.T) {
	predicate := "(> x 10)"
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithPredicate(api.ScriptLangAle, predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.Predicate)
	assert.Equal(t, api.ScriptLangAle, step.Predicate.Language)
	assert.Equal(t, predicate, step.Predicate.Script)
}

func TestWithAlePredicate(t *testing.T) {
	predicate := "(> x 10)"
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithAlePredicate(predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.Predicate)
	assert.Equal(t, api.ScriptLangAle, step.Predicate.Language)
	assert.Equal(t, predicate, step.Predicate.Script)
}

func TestWithLuaPredicate(t *testing.T) {
	predicate := "return x > 10"
	step, err := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithLuaPredicate(predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step.Predicate)
	assert.Equal(t, api.ScriptLangLua, step.Predicate.Language)
	assert.Equal(t, predicate, step.Predicate.Script)
}

func TestBuildValidHTTPStep(t *testing.T) {
	step, err := testClient().NewStep("Test Step").
		WithEndpoint("http://example.com/step").
		Required("input", api.TypeString).
		Output("output", api.TypeString).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, api.StepID("test-step"), step.ID)
}

func TestBuildValidScriptStep(t *testing.T) {
	step, err := testClient().NewStep("Script Step").
		WithScript("{:result 42}").
		Required("input", api.TypeString).
		Output("result", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, step)
	assert.Equal(t, api.StepTypeScript, step.Type)
}

func TestBuildInvalidStep(t *testing.T) {
	_, err := testClient().NewStep("").Build()

	assert.Error(t, err)
}

func TestChaining(t *testing.T) {
	step, err := testClient().NewStep("Chained Step").
		WithEndpoint("http://example.com/step").
		WithHealthCheck("http://example.com/health").
		WithTimeout(45*api.Second).
		WithVersion("2.1.0").
		Required("req1", api.TypeString).
		Required("req2", api.TypeNumber).
		Optional("opt1", api.TypeBoolean, "").
		Output("out1", api.TypeString).
		Output("out2", api.TypeNumber).
		WithAsyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, step.Type)
	requiredArgs := step.GetRequiredArgs()
	optionalArgs := step.GetOptionalArgs()
	outputArgs := step.GetOutputArgs()
	assert.Len(t, requiredArgs, 2)
	assert.Len(t, optionalArgs, 1)
	assert.Len(t, outputArgs, 2)
	assert.Equal(t, 45*api.Second, step.HTTP.Timeout)
	assert.Equal(t, "2.1.0", step.Version)
}

func TestImmutability(t *testing.T) {
	original := testClient().NewStep("Test Step")

	modified := original.
		WithID("custom-id").
		WithVersion("2.0.0").
		Required("input", api.TypeString).
		Output("output", api.TypeString).
		WithEndpoint("http://example.com").
		WithTimeout(60 * api.Second)

	_, err1 := original.Build()
	assert.Error(t, err1)

	step2, err2 := modified.Build()
	assert.NoError(t, err2)
	assert.NotNil(t, step2)

	assert.Equal(t, api.StepID("custom-id"), step2.ID)
	assert.Equal(t, "2.0.0", step2.Version)
	assert.Len(t, step2.GetRequiredArgs(), 1)
	assert.Len(t, step2.GetOutputArgs(), 1)
}

func TestImmutabilityMaps(t *testing.T) {
	base := testClient().NewStep("Test").WithEndpoint("http://example.com")

	builder1 := base.Required("arg1", api.TypeString)
	builder2 := base.Required("arg2", api.TypeNumber)

	step1, err1 := builder1.Build()
	assert.NoError(t, err1)
	requiredArgs1 := step1.GetRequiredArgs()
	assert.Len(t, requiredArgs1, 1)
	assert.Contains(t, requiredArgs1, api.Name("arg1"))
	assert.NotContains(t, requiredArgs1, api.Name("arg2"))

	step2, err2 := builder2.Build()
	assert.NoError(t, err2)
	requiredArgs2 := step2.GetRequiredArgs()
	assert.Len(t, requiredArgs2, 1)
	assert.Contains(t, requiredArgs2, api.Name("arg2"))
	assert.NotContains(t, requiredArgs2, api.Name("arg1"))
}

func TestImmutabilityHTTPConfig(t *testing.T) {
	base := testClient().NewStep("Test").
		WithEndpoint("http://example.com").
		WithHealthCheck("http://example.com/health")

	modified := base.WithEndpoint("http://different.com")

	step1, err1 := base.Build()
	assert.NoError(t, err1)
	assert.Equal(t, "http://example.com", step1.HTTP.Endpoint)
	assert.Equal(t, "http://example.com/health", step1.HTTP.HealthCheck)

	step2, err2 := modified.Build()
	assert.NoError(t, err2)
	assert.Equal(t, "http://different.com", step2.HTTP.Endpoint)
	assert.Equal(t, "http://example.com/health", step2.HTTP.HealthCheck)
}

func TestBuildValidationErrors(t *testing.T) {
	t.Run("missing_endpoint_for_http_step", func(t *testing.T) {
		_, err := testClient().NewStep("Test").
			Build()
		assert.ErrorIs(t, err, api.ErrHTTPRequired)
	})

	t.Run("missing_script_for_script_step", func(t *testing.T) {
		_, err := testClient().NewStep("Test").
			WithScriptExecution().
			Build()
		assert.ErrorIs(t, err, api.ErrScriptRequired)
	})

	t.Run("invalid_attribute_spec", func(t *testing.T) {
		_, err := testClient().NewStep("Test").
			WithEndpoint("http://example.com").
			Required("", api.TypeString).
			Build()
		assert.Error(t, err)
	})

	t.Run("valid_script_step", func(t *testing.T) {
		step, err := testClient().NewStep("Test").
			WithScript("(+ 1 2)").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, step.Script)
		assert.Equal(t, api.StepTypeScript, step.Type)
	})

	t.Run("ale_predicate", func(t *testing.T) {
		step, err := testClient().NewStep("Test").
			WithEndpoint("http://example.com").
			WithAlePredicate("(> count 10)").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, step.Predicate)
		assert.Equal(t, api.ScriptLangAle, step.Predicate.Language)
	})

	t.Run("lua_predicate", func(t *testing.T) {
		step, err := testClient().NewStep("Test").
			WithEndpoint("http://example.com").
			WithLuaPredicate("return count > 10").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, step.Predicate)
		assert.Equal(t, api.ScriptLangLua, step.Predicate.Language)
	})
}

func TestStepBuilderChaining(t *testing.T) {
	t.Run("complex_step_building", func(t *testing.T) {
		step, err := testClient().NewStep("Complex Step").
			WithID("complex").
			WithVersion("2.0.0").
			WithEndpoint("http://example.com/process").
			WithHealthCheck("http://example.com/health").
			WithTimeout(60000).
			Required("user_id", api.TypeString).
			Required("data", api.TypeObject).
			Optional("metadata", api.TypeObject, "{}").
			Output("result", api.TypeString).
			Output("status", api.TypeNumber).
			WithAsyncExecution().
			Build()

		assert.NoError(t, err)
		assert.Equal(t, api.StepID("complex"), step.ID)
		assert.Equal(t, "2.0.0", step.Version)
		assert.Equal(t, api.StepTypeAsync, step.Type)
		assert.Equal(t, int64(60000), step.HTTP.Timeout)
		assert.Len(t, step.Attributes, 5)
		assert.Equal(t, api.RoleRequired, step.Attributes["user_id"].Role)
		assert.Equal(t, api.RoleOptional, step.Attributes["metadata"].Role)
		assert.Equal(t, api.RoleOutput, step.Attributes["result"].Role)
	})

	t.Run("step_type_transitions", func(t *testing.T) {
		build := testClient().NewStep("Test")

		syncStep, err := build.
			WithEndpoint("http://example.com").
			WithSyncExecution().
			Build()
		assert.NoError(t, err)
		assert.Equal(t, api.StepTypeSync, syncStep.Type)

		asyncStep, err := build.
			WithEndpoint("http://example.com").
			WithAsyncExecution().
			Build()
		assert.NoError(t, err)
		assert.Equal(t, api.StepTypeAsync, asyncStep.Type)

		scriptStep, err := build.
			WithScript("(+ 1 2)").
			WithScriptExecution().
			Build()
		assert.NoError(t, err)
		assert.Equal(t, api.StepTypeScript, scriptStep.Type)
	})
}

func TestStepBuilderWithForEach(t *testing.T) {
	t.Run("for_each_attribute", func(t *testing.T) {
		step, err := testClient().NewStep("Batch Step").
			WithEndpoint("http://example.com").
			Required("users", api.TypeArray).
			Output("results", api.TypeArray).
			Build()

		assert.NoError(t, err)
		assert.Equal(t, api.TypeArray, step.Attributes["users"].Type)
	})
}

func testClient() *builder.Client {
	return builder.NewClient("http://localhost:8080", 30*time.Second)
}
