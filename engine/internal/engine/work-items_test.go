package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestForEachAggregatesOutputs(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "foreach-step",
			Name: "For Each Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:  api.RoleRequired,
					Type:  api.TypeArray,
					Input: &api.InputConfig{ForEach: true},
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeString,
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach", func() {
			err := env.Engine.StartFlow("wf-foreach", pl,
				flow.WithInit(api.InitArgs{"item": {[]any{"a", "b"}}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		attrs := fl.GetAttributes()
		results, ok := attrs["result"].([]map[string]any)
		assert.True(t, ok)
		assert.Len(t, results, 2)

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

func TestForEachTypedSlice(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "foreach-typed",
			Name: "For Each Typed",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:  api.RoleRequired,
					Type:  api.TypeArray,
					Input: &api.InputConfig{ForEach: true},
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeString,
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach-typed", func() {
			err := env.Engine.StartFlow("wf-foreach-typed", pl,
				flow.WithInit(api.InitArgs{"item": {[]string{"a", "b"}}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		attrs := fl.GetAttributes()
		results, ok := attrs["result"].([]map[string]any)
		assert.True(t, ok)
		assert.Len(t, results, 2)

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

func TestForEachTypedNumbers(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "foreach-nums",
			Name: "For Each Numbers",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:  api.RoleRequired,
					Type:  api.TypeArray,
					Input: &api.InputConfig{ForEach: true},
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeString,
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach-nums", func() {
			err := env.Engine.StartFlow("wf-foreach-nums", pl,
				flow.WithInit(api.InitArgs{"item": {[]int{2, 3}}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		attrs := fl.GetAttributes()
		results, ok := attrs["result"].([]map[string]any)
		assert.True(t, ok)
		assert.Len(t, results, 2)

		assertContainsEntry(t, results, map[string]any{
			"item":   float64(2),
			"result": "ok",
		})
		assertContainsEntry(t, results, map[string]any{
			"item":   float64(3),
			"result": "ok",
		})
	})
}

func TestOutputMappingDescendants(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
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

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{
			"payload": map[string]any{
				"sections": []any{
					map[string]any{"book": "A"},
					map[string]any{"book": "B"},
				},
			},
		})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-desc-mapping", func() {
			err := env.Engine.StartFlow("wf-desc-mapping", pl,
				flow.WithInit(api.InitArgs{"input": {"value"}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		raw := fl.Attributes["books"][0].Value
		books, ok := raw.([]any)
		assert.True(t, ok)
		assert.Equal(t, []any{"A", "B"}, books)
	})
}

func TestTooManyWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   "foreach-overload",
			Name: "ForEach Overload",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
			},
			Attributes: api.AttributeSpecs{
				"x": {
					Role:  api.RoleRequired,
					Type:  api.TypeArray,
					Input: &api.InputConfig{ForEach: true},
				},
				"y": {
					Role:  api.RoleRequired,
					Type:  api.TypeArray,
					Input: &api.InputConfig{ForEach: true},
				},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		// 101 * 101 = 10,201 items, exceeds MaxWorkItemsPerStep (10,000)
		xArr := make([]any, 101)
		yArr := make([]any, 101)
		for i := range xArr {
			xArr[i] = i
			yArr[i] = i
		}

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		err := env.Engine.StartFlow("wf-too-many", pl,
			flow.WithInit(api.InitArgs{"x": {xArr}, "y": {yArr}}),
		)
		assert.True(t, errors.Is(err, engine.ErrTooManyWorkItems))
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
