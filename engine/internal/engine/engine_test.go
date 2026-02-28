package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
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
				name: "catalog store",
				edit: func(deps *engine.Dependencies) {
					deps.CatalogStore = nil
				},
			},
			{
				name: "partition store",
				edit: func(deps *engine.Dependencies) {
					deps.PartitionStore = nil
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
		state, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Steps)
	})
}

func TestGetPartitionState(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state, err := eng.GetPartitionState()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Health)
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

func TestGetPartitionStateSeq(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state, seq, err := eng.GetPartitionStateSeq()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Health)
		assert.True(t, seq >= 0)
	})
}

func TestEngineStopGraceful(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		err := eng.Stop()
		assert.NoError(t, err)
	})
}
