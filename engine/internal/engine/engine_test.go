package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNew(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		assert.NotNil(t, eng)
	})
}

func TestNewMissingDependency(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		tests := []struct {
			name string
			edit func(*engine.Dependencies)
		}{
			{
				name: "engine store",
				edit: func(deps *engine.Dependencies) {
					deps.EngineStore = nil
				},
			},
			{
				name: "flow store",
				edit: func(deps *engine.Dependencies) {
					deps.FlowStore = nil
				},
			},
			{
				name: "step client",
				edit: func(deps *engine.Dependencies) {
					deps.StepClient = nil
				},
			},
			{
				name: "event hub",
				edit: func(deps *engine.Dependencies) {
					deps.EventHub = nil
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				deps := env.Dependencies()
				tt.edit(&deps)

				eng, err := engine.New(config.NewDefaultConfig(), deps)
				assert.Nil(t, eng)
				assert.Error(t, err)
				assert.True(t, errors.Is(err, engine.ErrMissingDependency))
			})
		}
	})
}

func TestNewInvalidConfig(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Config.APIPort = 0

		eng, err := env.NewEngineInstance()
		assert.Nil(t, eng)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, engine.ErrInvalidConfig))
		assert.True(t, errors.Is(err, config.ErrInvalidAPIPort))
	})
}

func TestNewDefaultsTimeDeps(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		deps := env.Dependencies()
		deps.Clock = nil
		deps.TimerConstructor = nil

		eng, err := engine.New(env.Config, deps)
		assert.NoError(t, err)
		assert.NotNil(t, eng)
	})
}

func TestStartStop(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		err := eng.Stop()
		assert.NoError(t, err)
	})
}

func TestGetCatalogState(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.NotNil(t, st)
		assert.NotNil(t, st.Steps)
	})
}

func TestGetClusterState(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st, err := eng.GetClusterState()
		assert.NoError(t, err)
		assert.NotNil(t, st)
		assert.NotNil(t, st.Nodes)
	})
}

func TestGetCatalogStateSeq(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state, seq, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Steps)
		assert.True(t, seq >= 0)
	})
}

func TestGetClusterStateSeq(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state, seq, err := eng.GetClusterStateSeq()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Nodes)
		assert.True(t, seq >= 0)
	})
}

func TestClusterTracksMultipleNodes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := *env.Config
		cfg.Raft.LocalID = "node-2"
		cfg.Raft.Servers = []raft.Server{
			{ID: "node-2", Address: "127.0.0.1:9702"},
		}

		peer, err := engine.New(&cfg, env.Dependencies())
		assert.NoError(t, err)
		if peer != nil {
			defer func() { _ = peer.Stop() }()
		}
		if !assert.NotNil(t, peer) {
			return
		}

		assert.NoError(t,
			peer.UpdateStepHealth("step-1", api.HealthHealthy, ""),
		)

		st, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		assert.Contains(t, st.Nodes, api.NodeID("node-2"))
		assert.Equal(t,
			api.HealthHealthy,
			st.Nodes["node-2"].Health["step-1"].Status,
		)
	})
}

func TestEngineStopGraceful(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		err := eng.Stop()
		assert.NoError(t, err)
	})
}

func TestExecPublishesEvents(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := helpers.NewSimpleStep("wrapper-step")
		env.MockClient.SetResponse(step.ID, api.Args{})

		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()
		w := wait.On(t, consumer)

		assert.NoError(t, env.Engine.RegisterStep(step))
		w.ForEvent(wait.EngineEvent(api.EventTypeStepRegistered))

		assert.NoError(t,
			env.Engine.UpdateStepHealth(step.ID, api.HealthHealthy, ""),
		)
		w.ForEvent(wait.EngineEvent(api.EventTypeStepHealthChanged))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}
		assert.NoError(t, env.Engine.StartFlow("wrapper-flow", pl))
		w.ForEvent(wait.FlowStarted("wrapper-flow"))
	})
}
