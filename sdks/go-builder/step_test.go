package builder_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdks/go-builder"
)

func TestNewStep(t *testing.T) {
	name := api.Name("Test Step")
	client := testClient()

	st, err := client.NewStep().WithName(name).
		WithEndpoint("http://example.com").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepID("test-step"), st.ID)
	assert.Equal(t, name, st.Name)
	assert.Equal(t, api.StepTypeSync, st.Type)
	assert.Equal(t, int64(30000), st.HTTP.Timeout)
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
			st, err := testClient().NewStep().WithName(tt.stepName).
				WithEndpoint("http://example.com").
				Build()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedID, st.ID)
		})
	}
}

func TestWithID(t *testing.T) {
	customID := "custom-id"
	st, err := testClient().NewStep().WithName("Test Step").
		WithID(customID).
		WithEndpoint("http://example.com").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepID(customID), st.ID)
	assert.Equal(t, api.Name("Test Step"), st.Name)
}

func TestStepWithEmptyLabels(t *testing.T) {
	st := testClient().NewStep().WithName("Test")

	assert.Equal(t, st, st.WithLabels(nil))
	assert.Equal(t, st, st.WithLabels(api.Labels{}))
}

func TestWithNameDoesNotOverrideID(t *testing.T) {
	st, err := testClient().NewStep().
		WithID("custom-id").
		WithName("Test Step").
		WithEndpoint("http://example.com").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepID("custom-id"), st.ID)
	assert.Equal(t, api.Name("Test Step"), st.Name)
}

func TestRequiredArg(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		Required("input1", api.TypeString).
		Required("input2", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.Len(t, st.Attributes, 2)
	assert.Contains(t, st.Attributes, api.Name("input1"))
	assert.EqualValues(t, api.TypeString, st.Attributes["input1"].Type)
	assert.EqualValues(t, api.RoleRequired, st.Attributes["input1"].Role)
}

func TestOptionalArg(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		Optional("optional1", api.TypeString, "").
		Optional("optional2", api.TypeNumber, "42").
		Build()

	assert.NoError(t, err)
	assert.Len(t, st.Attributes, 2)
	assert.Contains(t, st.Attributes, api.Name("optional1"))
	assert.EqualValues(t, api.RoleOptional, st.Attributes["optional1"].Role)
	assert.EqualValues(t, "42", st.Attributes["optional2"].Input.Default)
}

func TestConstArg(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		Const("const1", api.TypeString, `"fixed"`).
		Build()

	assert.NoError(t, err)
	assert.Len(t, st.Attributes, 1)
	assert.Contains(t, st.Attributes, api.Name("const1"))
	assert.EqualValues(t, api.RoleConst, st.Attributes["const1"].Role)
	assert.EqualValues(t, `"fixed"`, st.Attributes["const1"].Const.Value)
}

func TestOutputArg(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		Output("output1", api.TypeString).
		Output("output2", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.Len(t, st.Attributes, 2)
	assert.Contains(t, st.Attributes, api.Name("output1"))
	assert.EqualValues(t, api.TypeString, st.Attributes["output1"].Type)
	assert.EqualValues(t, api.RoleOutput, st.Attributes["output1"].Role)
}

func TestWithEndpoint(t *testing.T) {
	endpoint := "http://example.com/step"
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint(endpoint).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.HTTP)
	assert.Equal(t, endpoint, st.HTTP.Endpoint)
	assert.Equal(t, api.StepTypeSync, st.Type)
}

func TestWithMethod(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com/step").
		WithMethod("get").
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.HTTP)
	assert.Equal(t, "GET", st.HTTP.Method)
	assert.Equal(t, api.StepTypeSync, st.Type)
}

func TestWithMethodInvalid(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com/step").
		WithMethod("patch").
		Build()

	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestWithScript(t *testing.T) {
	script := "{:result (+ 1 2)}"
	st, err := testClient().NewStep().WithName("Test").
		WithScript(script).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.Script)
	assert.Equal(t, script, st.Script.Script)
	assert.Equal(t, api.ScriptLangAle, st.Script.Language)
	assert.Equal(t, api.StepTypeScript, st.Type)
}

func TestWithScriptLanguage(t *testing.T) {
	script := "custom script"
	lang := api.ScriptLangLua
	st, err := testClient().NewStep().WithName("Test").
		WithScriptLanguage(lang, script).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.Script)
	assert.Equal(t, script, st.Script.Script)
	assert.Equal(t, lang, st.Script.Language)
	assert.Equal(t, api.StepTypeScript, st.Type)
}

func TestWithScriptLanguageInvalid(t *testing.T) {
	script := "custom script"
	st, err := testClient().NewStep().WithName("Test").
		WithScriptLanguage("custom-lang", script).
		Build()

	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestWithHealthCheck(t *testing.T) {
	healthCheck := "http://example.com/health"
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com/step").
		WithHealthCheck(healthCheck).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.HTTP)
	assert.Equal(t, healthCheck, st.HTTP.HealthCheck)
}

func TestWithTimeout(t *testing.T) {
	timeout := api.Minute
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithTimeout(timeout).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, timeout, st.HTTP.Timeout)
}

func TestWithType(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithType(api.StepTypeAsync).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, st.Type)
}

func TestWithAsyncExecution(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithAsyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, st.Type)
}

func TestWithSyncExecution(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithSyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeSync, st.Type)
}

func TestWithScriptExecution(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test").
		WithScript("{:result 42}").
		WithScriptExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeScript, st.Type)
}

func TestWithFlowGoals(t *testing.T) {
	st, err := testClient().NewStep().WithName("Flow Step").
		WithFlowGoals("goal-a", "goal-b").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeFlow, st.Type)
	assert.Equal(t, []api.StepID{"goal-a", "goal-b"}, st.Flow.Goals)
}

func TestWithPredicate(t *testing.T) {
	predicate := "(> x 10)"
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithPredicate(api.ScriptLangAle, predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.Predicate)
	assert.Equal(t, api.ScriptLangAle, st.Predicate.Language)
	assert.Equal(t, predicate, st.Predicate.Script)
}

func TestWithAlePredicate(t *testing.T) {
	predicate := "(> x 10)"
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithAlePredicate(predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.Predicate)
	assert.Equal(t, api.ScriptLangAle, st.Predicate.Language)
	assert.Equal(t, predicate, st.Predicate.Script)
}

func TestWithLuaPredicate(t *testing.T) {
	predicate := "return x > 10"
	st, err := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com").
		WithLuaPredicate(predicate).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st.Predicate)
	assert.Equal(t, api.ScriptLangLua, st.Predicate.Language)
	assert.Equal(t, predicate, st.Predicate.Script)
}

func TestBuildValidHTTPStep(t *testing.T) {
	st, err := testClient().NewStep().WithName("Test Step").
		WithEndpoint("http://example.com/step").
		Required("input", api.TypeString).
		Output("output", api.TypeString).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st)
	assert.Equal(t, api.StepID("test-step"), st.ID)
}

func TestBuildValidScriptStep(t *testing.T) {
	st, err := testClient().NewStep().WithName("Script Step").
		WithScript("{:result 42}").
		Required("input", api.TypeString).
		Output("result", api.TypeNumber).
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, st)
	assert.Equal(t, api.StepTypeScript, st.Type)
}

func TestBuildInvalidStep(t *testing.T) {
	_, err := testClient().NewStep().WithName("").Build()

	assert.Error(t, err)
}

func TestChaining(t *testing.T) {
	st, err := testClient().NewStep().WithName("Chained Step").
		WithEndpoint("http://example.com/step").
		WithHealthCheck("http://example.com/health").
		WithTimeout(45*api.Second).
		Required("req1", api.TypeString).
		Required("req2", api.TypeNumber).
		Optional("opt1", api.TypeBoolean, "").
		Output("out1", api.TypeString).
		Output("out2", api.TypeNumber).
		WithAsyncExecution().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, api.StepTypeAsync, st.Type)
	requiredArgs := st.GetRequiredArgs()
	optionalArgs := st.GetOptionalArgs()
	outputArgs := st.GetOutputArgs()
	assert.Len(t, requiredArgs, 2)
	assert.Len(t, optionalArgs, 1)
	assert.Len(t, outputArgs, 2)
	assert.Equal(t, 45*api.Second, st.HTTP.Timeout)
}

func TestImmutability(t *testing.T) {
	original := testClient().NewStep().WithName("Test Step")

	modified := original.
		WithID("custom-id").
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
	assert.Len(t, step2.GetRequiredArgs(), 1)
	assert.Len(t, step2.GetOutputArgs(), 1)
}

func TestImmutabilityMaps(t *testing.T) {
	base := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com")

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
	base := testClient().NewStep().WithName("Test").
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
		_, err := testClient().NewStep().WithName("Test").
			Build()
		assert.ErrorIs(t, err, api.ErrHTTPRequired)
	})

	t.Run("missing_script_for_script_step", func(t *testing.T) {
		_, err := testClient().NewStep().WithName("Test").
			WithScriptExecution().
			Build()
		assert.ErrorIs(t, err, api.ErrScriptRequired)
	})

	t.Run("invalid_attribute_spec", func(t *testing.T) {
		_, err := testClient().NewStep().WithName("Test").
			WithEndpoint("http://example.com").
			Required("", api.TypeString).
			Build()
		assert.Error(t, err)
	})

	t.Run("valid_script_step", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Test").
			WithScript("(+ 1 2)").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, st.Script)
		assert.Equal(t, api.StepTypeScript, st.Type)
	})

	t.Run("ale_predicate", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Test").
			WithEndpoint("http://example.com").
			WithAlePredicate("(> count 10)").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, st.Predicate)
		assert.Equal(t, api.ScriptLangAle, st.Predicate.Language)
	})

	t.Run("lua_predicate", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Test").
			WithEndpoint("http://example.com").
			WithLuaPredicate("return count > 10").
			Build()
		assert.NoError(t, err)
		assert.NotNil(t, st.Predicate)
		assert.Equal(t, api.ScriptLangLua, st.Predicate.Language)
	})
}

func TestStepBuilderChaining(t *testing.T) {
	t.Run("complex_step_building", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Complex Step").
			WithID("complex").
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
		assert.Equal(t, api.StepID("complex"), st.ID)
		assert.Equal(t, api.StepTypeAsync, st.Type)
		assert.Equal(t, int64(60000), st.HTTP.Timeout)
		assert.Len(t, st.Attributes, 5)
		assert.Equal(t, api.RoleRequired, st.Attributes["user_id"].Role)
		assert.Equal(t, api.RoleOptional, st.Attributes["metadata"].Role)
		assert.Equal(t, api.RoleOutput, st.Attributes["result"].Role)
	})

	t.Run("step_type_transitions", func(t *testing.T) {
		build := testClient().NewStep().WithName("Test")

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

		flowStep, err := build.
			WithFlowGoals("goal-a").
			Build()
		assert.NoError(t, err)
		assert.Equal(t, api.StepTypeFlow, flowStep.Type)
	})
}

func TestStepBuilderWithForEach(t *testing.T) {
	t.Run("for_each_attribute", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Batch Step").
			WithEndpoint("http://example.com").
			Required("users", api.TypeArray).
			WithForEach("users").
			Output("results", api.TypeArray).
			Build()

		assert.NoError(t, err)
		assert.Equal(t, api.TypeArray, st.Attributes["users"].Type)
		assert.True(t, st.Attributes["users"].Input.ForEach)
	})
}

func TestStepBuilderWithLabels(t *testing.T) {
	t.Run("with_label", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Labeled Step").
			WithEndpoint("http://example.com").
			WithLabel("team", "core").
			WithLabel("env", "dev").
			Build()

		assert.NoError(t, err)
		assert.Equal(t, api.Labels{"team": "core", "env": "dev"}, st.Labels)
	})

	t.Run("with_labels_clone", func(t *testing.T) {
		labels := api.Labels{"team": "core"}
		st, err := testClient().NewStep().WithName("Labeled Step").
			WithEndpoint("http://example.com").
			WithLabel("env", "dev").
			WithLabels(labels).
			Build()

		assert.NoError(t, err)
		assert.Equal(t, api.Labels{"env": "dev", "team": "core"}, st.Labels)

		labels["team"] = "other"
		assert.Equal(t, api.Labels{"env": "dev", "team": "core"}, st.Labels)
	})
}

func TestStepBuilderWithMemoizable(t *testing.T) {
	t.Run("set_memoizable", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Memoizable Step").
			WithEndpoint("http://example.com").
			WithMemoizable().
			Build()

		assert.NoError(t, err)
		assert.True(t, st.Memoizable)
	})

	t.Run("default_not_memoizable", func(t *testing.T) {
		st, err := testClient().NewStep().WithName("Regular Step").
			WithEndpoint("http://example.com").
			Build()

		assert.NoError(t, err)
		assert.False(t, st.Memoizable)
	})
}

func TestUpdate(t *testing.T) {
	st := testClient().NewStep().WithName("Test").
		WithEndpoint("http://example.com")

	updated := st.Update()

	assert.NotNil(t, updated)
}

func testClient() *builder.Client {
	return builder.NewClient("http://localhost:8080", 30*time.Second)
}
