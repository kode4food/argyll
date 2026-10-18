package engine_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
)

func TestForEachAggregatesOutputs(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "foreach-step",
			Name: "For Each Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach", func() {
			err := env.Engine.StartPlan("wf-foreach", pl,
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

func TestMappedFanOut(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   "mapped",
			Name: "mapped",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { result = item }",
			},
			Attributes: api.AttributeSpecs{
				"items": {
					Role: api.RoleRequired,
					Required: &api.RequiredConfig{
						ForEach: true,
						Mapping: &api.MappingConfig{Name: "item"},
					},
				},
				"result": {Role: api.RoleOutput},
			},
		}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.StartPlan("mapped", pl,
			flow.WithInit(api.InitArgs{"items": {[]any{1, 2, 3}}}),
		))
		fl, err := env.Engine.GetFlowState("mapped")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 3)
		for _, work := range ex.WorkItems {
			assert.NotContains(t, work.Inputs, api.Name("items"))
			assert.Equal(t, work.Inputs["item"], work.Outputs["result"])
		}
		assert.ElementsMatch(t, []map[string]any{
			{"item": 1, "result": 1},
			{"item": 2, "result": 2},
			{"item": 3, "result": 3},
		}, ex.Outputs["result"])
	})
}

func TestForEachTypedSlice(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "foreach-typed",
			Name: "For Each Typed",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach-typed", func() {
			err := env.Engine.StartPlan("wf-foreach-typed", pl,
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"item": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-foreach-nums", func() {
			err := env.Engine.StartPlan("wf-foreach-nums", pl,
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"books": {
					Role: api.RoleOutput,
					Type: api.TypeAny,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{
							Script: &api.ScriptConfig{
								Language: api.ScriptLangJPath,
								Script:   "$..book",
							},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-desc-mapping", func() {
			err := env.Engine.StartPlan("wf-desc-mapping", pl,
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

func TestForEachEmpty(t *testing.T) {
	for _, value := range []any{[]any{}, []string{}, []int{}} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				st := helpers.NewStepWithOutputs("empty-each", "result")
				st.Attributes["item"] = &api.AttributeSpec{
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
				}
				assert.NoError(t, env.Engine.RegisterStep(st))
				req := api.CreateFlowRequest{
					ID:    "empty-each",
					Goals: api.Goals{Steps: []api.StepID{st.ID}},
					Init:  api.InitArgs{"item": {value}},
				}
				assert.NoError(t, env.Engine.StartFlow(req))
				fl, err := env.Engine.GetFlowState(req.ID)
				assert.NoError(t, err)
				assert.Equal(t, api.FlowCompleted, fl.Status)
				ex := fl.Executions[st.ID]
				assert.Equal(t, api.StepCompleted, ex.Status)
				assert.Empty(t, ex.WorkItems)
				assert.False(t, fl.DeactivatedAt.IsZero())
				assert.Empty(t, env.MockClient.GetInvocations())
			})
		})
	}
}

func TestForEachEmptyProduct(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewStepWithOutputs("empty-product", "result")
		init := api.InitArgs{}
		for i := range strconv.IntSize {
			name := api.Name(fmt.Sprintf("item%d", i))
			st.Attributes[name] = &api.AttributeSpec{
				Role:     api.RoleRequired,
				Type:     api.TypeArray,
				Required: &api.RequiredConfig{ForEach: true},
			}
			init[name] = []any{[]any{0, 1}}
		}
		init["item0"] = []any{[]any{}}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t,
			env.Engine.StartPlan("empty-product", pl, flow.WithInit(init)))
		fl, err := env.Engine.GetFlowState("empty-product")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Empty(t, fl.Executions[st.ID].WorkItems)
		assert.Empty(t, env.MockClient.GetInvocations())
	})
}

func TestWorkItemCountOverflow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewStepWithOutputs("overflow-each", "result")
		init := api.InitArgs{}
		for i := range strconv.IntSize {
			name := api.Name(fmt.Sprintf("item%d", i))
			st.Attributes[name] = &api.AttributeSpec{
				Role:     api.RoleRequired,
				Type:     api.TypeArray,
				Required: &api.RequiredConfig{ForEach: true},
			}
			init[name] = []any{[]any{0, 1}}
		}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		err := env.Engine.StartPlan("overflow-each", pl, flow.WithInit(init))
		assert.ErrorIs(t, err, engine.ErrTooManyWorkItems)
		_, err = env.Engine.GetFlowState("overflow-each")
		assert.ErrorIs(t, err, api.ErrFlowNotFound)
		assert.Empty(t, env.MockClient.GetInvocations())
	})
}

func TestTooManyWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   "foreach-overload",
			Name: "ForEach Overload",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			},
			Attributes: api.AttributeSpecs{
				"x": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
				},
				"y": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		err := env.Engine.StartPlan("wf-too-many", pl,
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
