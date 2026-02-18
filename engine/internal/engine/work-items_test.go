package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestForEachAggregatesOutputs(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := &api.Step{
			ID:   "foreach-step",
			Name: "For Each Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:    api.RoleRequired,
					Type:    api.TypeArray,
					ForEach: true,
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeString,
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-foreach", func() {
			err := env.Engine.StartFlow("wf-foreach", plan,
				flowopt.WithInit(api.Args{"item": []any{"a", "b"}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		attrs := flow.GetAttributes()
		rawResults, ok := attrs["result"].([]any)
		assert.True(t, ok)
		assert.Len(t, rawResults, 2)

		results := make([]map[string]any, 0, len(rawResults))
		for _, r := range rawResults {
			entry, ok := r.(map[string]any)
			assert.True(t, ok)
			results = append(results, entry)
		}

		assertContainsEntry(t, results, map[string]any{
			"item":   "a",
			"result": "ok",
		})
		assertContainsEntry(t, results, map[string]any{
			"item":   "b",
			"result": "ok",
		})
	})
}

func TestOutputMappingCollectsDescendants(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := &api.Step{
			ID:   "mapped-descendants-step",
			Name: "Mapped Descendants Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"books": {
					Role: api.RoleOutput,
					Type: api.TypeAny,
					Mapping: &api.AttributeMapping{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$..book",
						},
					},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{
			"payload": map[string]any{
				"sections": []any{
					map[string]any{"book": "A"},
					map[string]any{"book": "B"},
				},
			},
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-desc-mapping", func() {
			err := env.Engine.StartFlow("wf-desc-mapping", plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		raw := flow.Attributes["books"].Value
		books, ok := raw.([]any)
		assert.True(t, ok)
		assert.Equal(t, []any{"A", "B"}, books)
	})
}

func assertContainsEntry(
	t *testing.T, entries []map[string]any, expected map[string]any,
) {
	t.Helper()
	found := false
	for _, entry := range entries {
		if entry["item"] == expected["item"] &&
			entry["result"] == expected["result"] {
			found = true
			break
		}
	}
	assert.True(t, found)
}
