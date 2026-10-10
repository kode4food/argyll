package tests

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestFallbackGoals(t *testing.T) {
	t.Run("every goal in a set is required", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())
			registerSteps(t, env,
				goalStep("and-a", nil, "ra"),
				goalStep("and-b", []api.Name{"ra"}, "rb"),
			)
			env.MockClient.SetResponse("and-a", api.Args{"ra": "a"})
			env.MockClient.SetError("and-b", errors.New("boom"))

			fl := startGoals(t, env, "fallback-and", api.Goals{
				Steps: []api.StepID{"and-a", "and-b"},
			}, api.InitArgs{})

			assert.Equal(t, api.FlowFailed, fl.Status)
			assert.Equal(t, api.StepCompleted, fl.Executions["and-a"].Status)
			assert.Equal(t, api.StepFailed, fl.Executions["and-b"].Status)
		})
	})

	t.Run("later sets short-circuit on success", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())
			registerSteps(t, env,
				goalStep("first", nil, "r1"),
				goalStep("second", nil, "r2"),
			)
			env.MockClient.SetResponse("first", api.Args{"r1": "one"})
			env.MockClient.SetResponse("second", api.Args{"r2": "two"})

			fl := startGoals(t, env, "fallback-short", api.Goals{
				Steps: []api.StepID{"first"},
				Else:  &api.Goals{Steps: []api.StepID{"second"}},
			}, api.InitArgs{})

			assert.Equal(t, api.FlowCompleted, fl.Status)
			assert.Equal(t, api.StepSkipped, fl.Executions["second"].Status)
			assert.False(t, env.MockClient.WasInvoked("second"))
		})
	})

	t.Run("fails only when every set is impossible", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())
			registerSteps(t, env,
				goalStep("bad-1", nil, "r1"),
				goalStep("bad-2", nil, "r2"),
			)
			env.MockClient.SetError("bad-1", errors.New("boom"))
			env.MockClient.SetError("bad-2", errors.New("boom"))

			fl := startGoals(t, env, "fallback-all-fail", api.Goals{
				Steps: []api.StepID{"bad-1"},
				Else:  &api.Goals{Steps: []api.StepID{"bad-2"}},
			}, api.InitArgs{})

			assert.Equal(t, api.FlowFailed, fl.Status)
			assert.Equal(t,
				[]api.StepID{"bad-1", "bad-2"}, env.MockClient.GetInvocations())
		})
	})

	t.Run("fallback reuses completed work", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())
			registerSteps(t, env,
				goalStep("shared", nil, "s"),
				goalStep("step1", []api.Name{"v2"}, "r1"),
				goalStep("step2", []api.Name{"s"}, "v2"),
				goalStep("step3", []api.Name{"s"}, "r3"),
				goalStep("step4", nil, "r4"),
			)
			env.MockClient.SetResponse("shared", api.Args{"s": "shared"})
			env.MockClient.SetResponse("step2", api.Args{"v2": "two"})
			env.MockClient.SetError("step1", errors.New("boom"))
			env.MockClient.SetResponse("step3", api.Args{"r3": "three"})
			env.MockClient.SetResponse("step4", api.Args{"r4": "four"})

			fl := startGoals(t, env, "fallback-reuse", api.Goals{
				Steps: []api.StepID{"step1", "step2"},
				Else: &api.Goals{
					Steps: []api.StepID{"step2", "step3"},
					Else:  &api.Goals{Steps: []api.StepID{"step4"}},
				},
			}, api.InitArgs{})

			assert.Equal(t, api.FlowCompleted, fl.Status)
			assert.Equal(t, api.StepFailed, fl.Executions["step1"].Status)
			assert.Equal(t, api.StepCompleted, fl.Executions["step2"].Status)
			assert.Equal(t, api.StepCompleted, fl.Executions["step3"].Status)
			assert.Equal(t, api.StepSkipped, fl.Executions["step4"].Status)
			assert.Equal(t, "two", fl.Attributes["v2"][0].Value)
			assert.Equal(t, "shared", fl.Attributes["s"][0].Value)

			calls := env.MockClient.GetInvocations()
			for _, sid := range []api.StepID{
				"shared", "step1", "step2", "step3",
			} {
				assert.Equal(t, 1, countOf(calls, sid))
			}
			assert.NotContains(t, calls, api.StepID("step4"))
			assert.Greater(t,
				slices.Index(calls, "step3"), slices.Index(calls, "step1"))
		})
	})

	t.Run("rolls back only abandoned work", func(t *testing.T) {
		for _, compensate := range []bool{true, false} {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				assert.NoError(t, env.Engine.Start())
				registerSteps(t, env,
					compensatingStep("shared", nil, "s"),
					compensatingStep("stock", []api.Name{"s"}, "reservation"),
					goalStep("notify", []api.Name{"reservation"}, "sent"),
					goalStep("degrade", []api.Name{"s"}, "degraded"),
				)
				env.MockClient.SetResponse("shared", api.Args{"s": "x"})
				env.MockClient.SetResponse("stock",
					api.Args{"reservation": "r"},
				)
				env.MockClient.SetError("notify", errors.New("boom"))
				env.MockClient.SetResponse("degrade", api.Args{"degraded": "y"})

				var mu sync.Mutex
				var undone []api.StepID
				record := func(s *api.Step, _ api.Args, _ api.Metadata) error {
					mu.Lock()
					defer mu.Unlock()
					undone = append(undone, s.ID)
					return nil
				}
				env.MockClient.SetCompensate("shared", record)
				env.MockClient.SetCompensate("stock", record)

				fid := api.FlowID("fallback-rollback")
				req := api.CreateFlowRequest{
					ID:         fid,
					Compensate: compensate,
					Goals: api.Goals{
						Steps: []api.StepID{"notify"},
						Else:  &api.Goals{Steps: []api.StepID{"degrade"}},
					},
				}
				env.WaitFor(wait.FlowDeactivated(fid), func() {
					assert.NoError(t, env.Engine.StartFlow(req))
				})

				fl, err := env.Engine.GetFlowState(fid)
				assert.NoError(t, err)
				assert.Equal(t, api.FlowCompleted, fl.Status)

				mu.Lock()
				defer mu.Unlock()
				if compensate {
					assert.Equal(t, []api.StepID{"stock"}, undone)
				} else {
					assert.Empty(t, undone)
				}
			})
		}
	})

	t.Run("required inputs cover every set", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())
			registerSteps(t, env,
				goalStep("primary", []api.Name{"primary-input"}, "p"),
				goalStep("backup", []api.Name{"fallback-input"}, "b"),
			)
			goals := api.Goals{
				Steps: []api.StepID{"primary"},
				Else:  &api.Goals{Steps: []api.StepID{"backup"}},
			}

			pl, err := env.Engine.CreatePlan(api.ExecutionPlanRequest{
				Goals: goals,
			})
			assert.NoError(t, err)
			assert.ElementsMatch(t,
				[]api.Name{"primary-input", "fallback-input"}, pl.Required)

			for _, init := range []api.InitArgs{
				{"primary-input": {"x"}},
				{"fallback-input": {"x"}},
			} {
				err := env.Engine.StartFlow(api.CreateFlowRequest{
					ID:    "fallback-missing",
					Goals: goals,
					Init:  init,
				})
				assert.ErrorIs(t, err, api.ErrRequiredInputs)
			}

			env.MockClient.SetResponse("primary", api.Args{"p": "ok"})
			fl := startGoals(t, env, "fallback-both", goals, api.InitArgs{
				"primary-input":  {"x"},
				"fallback-input": {"y"},
			})
			assert.Equal(t, api.FlowCompleted, fl.Status)
		})
	})
}

func startGoals(
	t *testing.T, env *helpers.TestEngineEnv, fid api.FlowID,
	goals api.Goals, init api.InitArgs,
) api.FlowState {
	t.Helper()
	return env.WaitForFlowStatus(fid, func() {
		err := env.Engine.StartFlow(api.CreateFlowRequest{
			ID:    fid,
			Goals: goals,
			Init:  init,
		})
		assert.NoError(t, err)
	})
}

func registerSteps(
	t *testing.T, env *helpers.TestEngineEnv, steps ...*api.Step,
) {
	t.Helper()
	for _, st := range steps {
		assert.NoError(t, env.Engine.RegisterStep(st))
	}
}

func goalStep(
	id api.StepID, inputs []api.Name, outputs ...api.Name,
) *api.Step {
	st := helpers.NewStepWithOutputs(id, outputs...)
	for _, name := range inputs {
		st.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
	}
	return st
}

func compensatingStep(
	id api.StepID, inputs []api.Name, outputs ...api.Name,
) *api.Step {
	st := goalStep(id, inputs, outputs...)
	st.Handling = api.HandlingCompensated
	st.HTTP.Compensate = &api.HTTPAction{Endpoint: "http://test:8080/undo"}
	return st
}

func countOf(ids []api.StepID, sid api.StepID) int {
	n := 0
	for _, id := range ids {
		if id == sid {
			n++
		}
	}
	return n
}
