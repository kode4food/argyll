package engine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/flow"
	"github.com/kode4food/argyll/engine/pkg/plan"
)

type atomicFlowBackend struct {
	timebox.Backend
	mu            sync.Mutex
	childConflict bool
	conflicts     map[api.EventType]bool
	commits       map[api.EventType][]timebox.AppendRequest
}

func TestStartDuplicate(t *testing.T) {
	t.Run("same goals is idempotent", func(t *testing.T) {
		helpers.WithStartedEngine(t, func(eng *engine.Engine) {
			st := helpers.NewSimpleStep("step-1")

			err := eng.RegisterStep(st)
			assert.NoError(t, err)

			pl := &api.ExecutionPlan{
				Goals: api.Goals{Steps: []api.StepID{"step-1"}},
				Steps: api.Steps{st.ID: st},
			}

			err = eng.StartPlan("wf-dup", pl)
			assert.NoError(t, err)

			err = eng.StartPlan("wf-dup", pl)
			assert.NoError(t, err)
		})
	})

	t.Run("different goals conflicts", func(t *testing.T) {
		helpers.WithStartedEngine(t, func(eng *engine.Engine) {
			st1 := helpers.NewSimpleStep("step-1")
			st2 := helpers.NewSimpleStep("step-2")

			assert.NoError(t, eng.RegisterStep(st1))
			assert.NoError(t, eng.RegisterStep(st2))

			pl1 := &api.ExecutionPlan{
				Goals: api.Goals{Steps: []api.StepID{"step-1"}},
				Steps: api.Steps{st1.ID: st1},
			}
			pl2 := &api.ExecutionPlan{
				Goals: api.Goals{Steps: []api.StepID{"step-2"}},
				Steps: api.Steps{st2.ID: st2},
			}

			err := eng.StartPlan("wf-conflict", pl1)
			assert.NoError(t, err)

			err = eng.StartPlan("wf-conflict", pl2)
			assert.ErrorIs(t, err, api.ErrFlowExists)
		})
	})
}

func TestStartFlowSchedulesWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-start")
		st.HTTP.Invoke.Mode = api.ActionModeAsync
		st.HTTP.Invoke.Timeout = 30 * api.Second

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("_")
		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err = env.Engine.StartPlan(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepActive, ex.Status)
		assert.Len(t, ex.WorkItems, 1)
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkActive, item.Status)
		}
	})
}

func TestStartMissingInput(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("step-needs-input")
		st.Attributes["required_value"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		pl := &api.ExecutionPlan{
			Goals:    api.Goals{Steps: []api.StepID{"step-needs-input"}},
			Steps:    api.Steps{st.ID: st},
			Required: []api.Name{"required_value"},
		}

		err := eng.StartPlan("wf-missing", pl)
		assert.Error(t, err)
	})
}

func TestStartRejectsPartialParent(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("step-parent-meta")
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		err := eng.StartPlan("wf-partial-parent-meta", pl,
			flow.WithMetadata(api.Metadata{
				api.MetaParentFlowID: "parent",
			}),
		)
		assert.ErrorContains(t, err, "partial parent metadata")
	})
}

func TestStartFlowSimple(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:   "goal-step",
			Name: "Goal",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

		pl := &api.ExecutionPlan{
			Goals:    api.Goals{Steps: []api.StepID{"goal-step"}},
			Required: []api.Name{},
			Steps: api.Steps{
				"goal-step": st,
			},
		}

		err = env.Engine.StartPlan("wf-simple", pl)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState("wf-simple")
		assert.NoError(t, err)
		assert.NotNil(t, fl)
		assert.Equal(t, api.FlowID("wf-simple"), fl.ID)
	})
}

func TestStartChildFlowUsesPlan(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		child := helpers.NewSimpleStep("child")
		child.HTTP.Invoke.Mode = api.ActionModeAsync
		parent := &api.Step{
			ID:   "sub",
			Name: "Sub Flow",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))
		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    env.Engine.Matcher,
			Children: env.Engine.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
		})
		assert.NoError(t, err)
		assert.NoError(t, env.Engine.StartPlan("parent", pl,
			flow.WithMetadata(api.Metadata{"source": "test"}),
		))

		updated := helpers.NewSimpleStep(child.ID)
		updated.HTTP.Invoke.Mode = api.ActionModeAsync
		updated.Attributes["new-input"] = &api.AttributeSpec{
			Role: api.RoleRequired, Type: api.TypeString,
		}
		assert.NoError(t, env.Engine.UpdateStep(updated))
		assert.NoError(t, env.Engine.Start())
		fl := helpers.WaitForFlowState(t, env.Engine, helpers.FlowStateQuery{
			FlowID: "parent", Timeout: wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				for _, work := range fl.Executions[parent.ID].WorkItems {
					return work.Status == api.WorkActive
				}
				return false
			},
		})
		for tkn := range fl.Executions[parent.ID].WorkItems {
			fid := api.FlowID("parent:sub:" + tkn)
			child, err := env.Engine.GetFlowState(fid)
			assert.NoError(t, err)
			assert.Empty(t, child.Plan.Required)
			assert.NotContains(t, child.Plan.Steps["child"].Attributes,
				api.Name("new-input"))
			assert.Equal(t, "test", child.Metadata["source"])
			assert.Equal(t, api.FlowID("parent"),
				child.Metadata[api.MetaParentFlowID])
			assert.Equal(t, parent.ID, child.Metadata[api.MetaParentStepID])
			assert.Equal(t, tkn, child.Metadata[api.MetaParentWorkItemToken])
		}
	})
}

func TestCreatePlanEmbedsChildPlans(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		child := &api.Step{
			ID:   "child",
			Name: "Child",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test",
					Timeout:  30 * api.Second,
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(child))

		parent := &api.Step{
			ID:   "parent",
			Name: "Parent",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{child.ID}},
			},
			Attributes: api.AttributeSpecs{
				"wrapped": {
					Role: api.RoleOutput,
					Type: api.TypeString,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "result"},
					},
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(parent))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		pl, err := plan.Create(&plan.Request{
			Match:    eng.Matcher,
			Children: eng.Children,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		if assert.Contains(t, pl.Children, parent.ID) {
			childPlan := pl.Children[parent.ID]
			assert.NotNil(t, childPlan)
			assert.Contains(t, childPlan.Steps, child.ID)
		}
	})
}

func TestCreatePlanRestrictsChildFlowToSpace(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		provider := helpers.NewSimpleStep("provider")
		provider.Attributes = api.AttributeSpecs{
			"input": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(provider))

		goal := helpers.NewSimpleStep("goal")
		goal.Tags = api.Tags{"domain:payments"}
		goal.Attributes = api.AttributeSpecs{
			"input": {Role: api.RoleRequired, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(goal))

		sp := api.Space{
			ID:   "payments",
			Name: "Payments",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		}
		assert.NoError(t, eng.RegisterSpace(sp))

		parent := &api.Step{
			ID:         "parent",
			Name:       "Parent",
			Type:       api.StepTypeFlow,
			Attributes: api.AttributeSpecs{},
			Flow: &api.FlowConfig{
				Goals:   api.Goals{Steps: []api.StepID{goal.ID}},
				SpaceID: sp.ID,
			},
		}
		assert.NoError(t, eng.RegisterStep(parent))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    eng.Matcher,
			Children: eng.Children,
			Catalog:  cat,
			Steps:    cat.Steps,
			Goals:    api.Goals{Steps: []api.StepID{parent.ID}},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		child := pl.Children[parent.ID]
		assert.NotContains(t, child.Steps, provider.ID)
		assert.Contains(t, child.Required, api.Name("input"))
	})
}

func TestAtomicFlowLifecycle(t *testing.T) {
	for _, childConflict := range []bool{false, true} {
		name := "parent conflict"
		if childConflict {
			name = "child conflict"
		}
		t.Run(name, func(t *testing.T) {
			backend := &atomicFlowBackend{
				Backend:       memory.Open(),
				childConflict: childConflict,
				conflicts:     map[api.EventType]bool{},
				commits:       map[api.EventType][]timebox.AppendRequest{},
			}
			helpers.WithTestBackend(t, backend,
				func(env *helpers.TestEngineEnv) {
					leaf := helpers.NewSimpleStep("leaf")
					leaf.HTTP.Invoke.Mode = api.ActionModeAsync
					sibling := helpers.NewSimpleStep("sibling")
					sibling.HTTP.Invoke.Mode = api.ActionModeAsync
					sub := &api.Step{ID: "sub", Type: api.StepTypeFlow}
					childPlan := &api.ExecutionPlan{
						Goals: api.Goals{Steps: []api.StepID{leaf.ID}},
						Steps: api.Steps{leaf.ID: leaf},
					}
					pl := &api.ExecutionPlan{
						Goals: api.Goals{
							Steps: []api.StepID{sub.ID, sibling.ID},
						},
						Steps: api.Steps{sub.ID: sub, sibling.ID: sibling},
						Children: map[api.StepID]*api.ExecutionPlan{
							sub.ID: childPlan,
						},
					}
					leafStarted := make(chan api.Token, 1)
					siblingStarted := make(chan api.Token, 1)
					env.MockClient.SetInvoke(leaf.ID,
						func(
							_ *api.Step, _ api.Args, meta api.Metadata,
						) (api.Args, error) {
							token, _ := meta.GetString[api.Token](
								api.MetaReceiptToken,
							)
							leafStarted <- token
							return api.Args{}, nil
						},
					)
					env.MockClient.SetInvoke(sibling.ID,
						func(
							_ *api.Step, _ api.Args, meta api.Metadata,
						) (api.Args, error) {
							token, _ := meta.GetString[api.Token](
								api.MetaReceiptToken,
							)
							siblingStarted <- token
							return api.Args{}, nil
						},
					)
					assert.NoError(t, env.Engine.StartPlan("atomic", pl))
					assert.NoError(t, env.Engine.Start())
					parent := helpers.WaitForFlowState(t, env.Engine,
						helpers.FlowStateQuery{
							FlowID:  "atomic",
							Timeout: wait.DefaultTimeout,
							Accept: func(fl api.FlowState) bool {
								ex := fl.Executions[sub.ID]
								for _, w := range ex.WorkItems {
									return w.Status == api.WorkActive
								}
								return false
							},
						},
					)
					var fid api.FlowID
					for tkn := range parent.Executions[sub.ID].WorkItems {
						fid = api.FlowID("atomic:sub:" + tkn)
					}
					child, err := env.Engine.GetFlowState(fid)
					assert.NoError(t, err)
					assert.Equal(t, api.FlowActive, child.Status)

					var token api.Token
					select {
					case token = <-leafStarted:
					case <-time.After(wait.DefaultTimeout):
						t.Fatal("child work did not start")
					}
					assert.NoError(t,
						env.Engine.CompleteWork(
							api.FlowStep{FlowID: fid, StepID: leaf.ID},
							token, api.Args{},
						),
					)
					child, err = env.Engine.GetFlowState(fid)
					assert.NoError(t, err)
					assert.Equal(t, api.FlowCompleted, child.Status)
					assert.True(t, child.DeactivatedAt.IsZero())

					select {
					case token = <-siblingStarted:
					case <-time.After(wait.DefaultTimeout):
						t.Fatal("sibling work did not start")
					}
					assert.NoError(t,
						env.Engine.CompleteWork(
							api.FlowStep{FlowID: "atomic", StepID: sibling.ID},
							token, api.Args{},
						),
					)
					child, err = env.Engine.GetFlowState(fid)
					assert.NoError(t, err)
					assert.False(t, child.DeactivatedAt.IsZero())

					backend.mu.Lock()
					defer backend.mu.Unlock()
					for _, phase := range []api.EventType{
						api.EventTypeFlowStarted,
						api.EventTypeFlowCompleted,
						api.EventTypeFlowDeactivated,
					} {
						assert.True(t, backend.conflicts[phase])
						batch := backend.commits[phase]
						kinds := map[api.FlowID][]api.EventType{}
						flowRequests := 0
						scheduleRequests := 0
						for _, req := range batch {
							if id, ok := events.ParseFlowID(req.ID); !ok {
								scheduleRequests++
							} else {
								flowRequests++
								for _, ev := range req.Events {
									typ := api.EventType(ev.Type)
									kinds[id] = append(kinds[id], typ)
								}
							}
						}
						assert.Equal(t, 2, flowRequests)
						assert.NotZero(t, scheduleRequests)
						assert.Contains(t, kinds[fid], phase)
						switch phase {
						case api.EventTypeFlowStarted:
							assert.Contains(t,
								kinds["atomic"], api.EventTypeWorkStarted)
						case api.EventTypeFlowCompleted:
							assert.Contains(t,
								kinds["atomic"], api.EventTypeWorkSucceeded)
						case api.EventTypeFlowDeactivated:
							assert.Contains(t, kinds["atomic"], phase)
						}
					}
				},
			)
		})
	}
}

func TestNestedFlowSettlement(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		leaf := helpers.NewSimpleStep("leaf")
		sub := &api.Step{ID: "sub", Type: api.StepTypeFlow}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{leaf.ID}},
			Steps: api.Steps{leaf.ID: leaf},
		}
		for range 2 {
			pl = &api.ExecutionPlan{
				Goals:    api.Goals{Steps: []api.StepID{sub.ID}},
				Steps:    api.Steps{sub.ID: sub},
				Children: map[api.StepID]*api.ExecutionPlan{sub.ID: pl},
			}
		}
		env.MockClient.SetInvoke(leaf.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				env.ConflictOnNextAppend("nested")
				return api.Args{}, nil
			},
		)
		assert.NoError(t, env.Engine.Start())
		env.WaitFor(wait.FlowDeactivated("nested"), func() {
			assert.NoError(t, env.Engine.StartPlan("nested", pl))
		})
		assert.True(t, env.ConflictFired())
		fid := api.FlowID("nested")
		for depth := range 3 {
			fl, err := env.Engine.GetFlowState(fid)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowCompleted, fl.Status)
			assert.False(t, fl.DeactivatedAt.IsZero())
			evs, err := env.Engine.GetFlowEvents(fid)
			assert.NoError(t, err)
			counts := map[api.EventType]int{}
			for _, ev := range evs {
				counts[api.EventType(ev.Type)]++
			}
			assert.Equal(t, 1, counts[api.EventTypeFlowCompleted])
			assert.Equal(t, 1, counts[api.EventTypeFlowDeactivated])
			if depth < 2 {
				for tkn := range fl.Executions[sub.ID].WorkItems {
					fid = api.FlowID(string(fid) + ":sub:" + string(tkn))
				}
			}
		}
		assert.Len(t, env.MockClient.GetInvocations(), 1)
	})
}

func (b *atomicFlowBackend) Append(reqs ...timebox.AppendRequest) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var phase api.EventType
	for _, req := range reqs {
		if req.ID == events.FlowKey("atomic") {
			continue
		}
		for _, ev := range req.Events {
			switch api.EventType(ev.Type) {
			case api.EventTypeFlowStarted, api.EventTypeFlowCompleted,
				api.EventTypeFlowDeactivated:
				phase = api.EventType(ev.Type)
			}
		}
	}
	for _, req := range reqs {
		child := req.ID != events.FlowKey("atomic")
		if phase != "" && !b.conflicts[phase] && child == b.childConflict {
			b.conflicts[phase] = true
			return &timebox.VersionConflictError{
				ID:               req.ID,
				ExpectedSequence: req.ExpectedSequence,
				ActualSequence:   req.ExpectedSequence + 1,
			}
		}
	}
	if err := b.Backend.Append(reqs...); err != nil {
		return err
	}
	if phase != "" {
		b.commits[phase] = reqs
	}
	return nil
}
