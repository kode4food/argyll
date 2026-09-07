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
)

type atomicFlowBackend struct {
	timebox.Backend
	mu            sync.Mutex
	childConflict bool
	conflicts     map[api.EventType]bool
	commits       map[api.EventType][]timebox.AppendRequest
}

func TestAtomicFlowLifecycle(t *testing.T) {
	for _, childConflict := range []bool{false, true} {
		name := "parent conflict"
		if childConflict {
			name = "child conflict"
		}
		t.Run(name, func(t *testing.T) {
			backend := &atomicFlowBackend{
				Backend:       memory.NewPersistence(),
				childConflict: childConflict,
				conflicts:     map[api.EventType]bool{},
				commits:       map[api.EventType][]timebox.AppendRequest{},
			}
			cfg := helpers.NewTestConfig()
			store, err := timebox.NewStore(backend, cfg.FlowStoreConfig())
			assert.NoError(t, err)
			helpers.WithTestEnvDeps(t,
				engine.Dependencies{FlowStore: store},
				func(env *helpers.TestEngineEnv) {
					leaf := helpers.NewSimpleStep("leaf")
					leaf.HTTP.Invoke.Mode = api.ActionModeAsync
					sibling := helpers.NewSimpleStep("sibling")
					sibling.HTTP.Invoke.Mode = api.ActionModeAsync
					sub := &api.Step{ID: "sub", Type: api.StepTypeFlow}
					childPlan := &api.ExecutionPlan{
						Goals: []api.StepID{leaf.ID},
						Steps: api.Steps{leaf.ID: leaf},
					}
					pl := &api.ExecutionPlan{
						Goals: []api.StepID{sub.ID, sibling.ID},
						Steps: api.Steps{sub.ID: sub, sibling.ID: sibling},
						Children: map[api.StepID]*api.ExecutionPlan{
							sub.ID: childPlan,
						},
					}
					leafStarted := make(chan api.Token, 1)
					siblingStarted := make(chan api.Token, 1)
					env.MockClient.SetHandler(leaf.ID,
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
					env.MockClient.SetHandler(sibling.ID,
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
					assert.NoError(t, env.Engine.StartFlow("atomic", pl))
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
					assert.Empty(t, leafStarted)

					// This store has no commit subscriber: explicitly wake it
					assert.NoError(t, env.Engine.RecoverFlow(fid))
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
						assert.Len(t, batch, 2)
						kinds := map[api.FlowID][]api.EventType{}
						for _, req := range batch {
							id, ok := events.ParseFlowID(req.ID)
							assert.True(t, ok)
							for _, ev := range req.Events {
								typ := api.EventType(ev.Type)
								kinds[id] = append(kinds[id], typ)
							}
						}
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
			Goals: []api.StepID{leaf.ID},
			Steps: api.Steps{leaf.ID: leaf},
		}
		for range 2 {
			pl = &api.ExecutionPlan{
				Goals:    []api.StepID{sub.ID},
				Steps:    api.Steps{sub.ID: sub},
				Children: map[api.StepID]*api.ExecutionPlan{sub.ID: pl},
			}
		}
		env.MockClient.SetHandler(leaf.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				env.ConflictOnNextAppend("nested")
				return api.Args{}, nil
			},
		)
		assert.NoError(t, env.Engine.Start())
		env.WaitFor(wait.FlowDeactivated("nested"), func() {
			assert.NoError(t, env.Engine.StartFlow("nested", pl))
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
