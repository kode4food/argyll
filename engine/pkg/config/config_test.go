package config_test

import (
	"testing"

	"github.com/kode4food/timebox"
	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestConfigValidation(t *testing.T) {
	as := assert.New(t)

	t.Run("valid_default_config", func(t *testing.T) {
		cfg := config.NewDefaultConfig()
		as.ConfigValid(cfg)
	})

	tests := []struct {
		name          string
		configMod     func(*config.Config)
		errorContains string
	}{
		{
			name: "zero_step_timeout",
			configMod: func(c *config.Config) {
				c.StepTimeout = 0
			},
			errorContains: "step timeout must be positive",
		},
		{
			name: "zero_retry_max_retries",
			configMod: func(c *config.Config) {
				c.Work.MaxRetries = 0
			},
			errorContains: "retry max retries cannot be zero",
		},
		{
			name: "zero_retry_backoff",
			configMod: func(c *config.Config) {
				c.Work.InitBackoff = 0
			},
			errorContains: "retry initial backoff must be positive",
		},
		{
			name: "zero_retry_max_backoff",
			configMod: func(c *config.Config) {
				c.Work.MaxBackoff = 0
			},
			errorContains: "retry max backoff must be positive",
		},
		{
			name: "retry_max_backoff_too_small",
			configMod: func(c *config.Config) {
				c.Work.InitBackoff = 1000
				c.Work.MaxBackoff = 999
			},
			errorContains: "retry max backoff must be >=",
		},
		{
			name: "invalid_retry_backoff_type",
			configMod: func(c *config.Config) {
				c.Work.BackoffType = "weird"
			},
			errorContains: "invalid retry backoff type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			tt.configMod(cfg)
			as.ConfigInvalid(cfg, tt.errorContains)
		})
	}
}

func TestDefaultConfigValues(t *testing.T) {
	as := assert.New(t)

	cfg := config.NewDefaultConfig()

	as.Equal(config.DefaultStepTimeout, cfg.StepTimeout)
	as.Equal(api.NodeID(config.DefaultNodeID), cfg.NodeID)
	as.Empty(cfg.Nodes)
	as.False(cfg.Timebox.TrimEvents)
	as.Equal(timebox.DefaultSnapshotRatio, cfg.Timebox.SnapshotRatio)
	as.Equal(config.DefaultTimeboxCacheSize, cfg.Timebox.CacheSize)
	as.Nil(cfg.Timebox.Indexer)

	engStore := cfg.EngineStoreConfig()
	as.True(engStore.TrimEvents)
	as.Equal(timebox.DefaultSnapshotRatio, engStore.SnapshotRatio)
	as.Equal(config.EngineStoreCacheSize, engStore.CacheSize)
	as.Nil(engStore.Indexer)

	flowStore := cfg.FlowStoreConfig()
	as.False(flowStore.TrimEvents)
	as.Equal(config.DefaultFlowSnapshotRatio, flowStore.SnapshotRatio)
	as.Equal(config.DefaultTimeboxCacheSize, flowStore.CacheSize)
	as.NotNil(flowStore.Indexer)
}

func TestDefaultTimebox(t *testing.T) {
	tb := config.DefaultTimebox()

	testify.False(t, tb.TrimEvents)
	testify.Equal(t, timebox.DefaultSnapshotRatio, tb.SnapshotRatio)
	testify.Equal(t, config.DefaultTimeboxCacheSize, tb.CacheSize)
	testify.Equal(t, timebox.DefaultMaxRetries, tb.MaxRetries)
	testify.Nil(t, tb.Indexer)
}

func TestStoreConfigs(t *testing.T) {
	cfg := config.NewDefaultConfig()

	engStore := cfg.EngineStoreConfig()
	testify.True(t, engStore.TrimEvents)
	testify.Equal(t, timebox.DefaultSnapshotRatio, engStore.SnapshotRatio)
	testify.Equal(t, config.EngineStoreCacheSize, engStore.CacheSize)
	testify.Nil(t, engStore.Indexer)

	flowStore := cfg.FlowStoreConfig()
	testify.False(t, flowStore.TrimEvents)
	testify.Equal(t, config.DefaultFlowSnapshotRatio, flowStore.SnapshotRatio)
	testify.Equal(t, config.DefaultTimeboxCacheSize, flowStore.CacheSize)
	testify.NotNil(t, flowStore.Indexer)
}

func TestValidateMinimumTimeout(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.StepTimeout = 1

	testify.NoError(t, cfg.Validate())
}

func TestWithDefaults(t *testing.T) {
	t.Run("fills zero work config", func(t *testing.T) {
		cfg := &config.Config{StepTimeout: 1000}
		out := cfg.WithWorkDefaults()

		testify.Equal(t,
			config.DefaultRetryMaxRetries, out.Work.MaxRetries,
		)
		testify.Equal(t,
			int64(config.DefaultRetryInitBackoff), out.Work.InitBackoff,
		)
		testify.Equal(t,
			int64(config.DefaultMaxRetryBackoff), out.Work.MaxBackoff,
		)
		testify.Equal(t,
			config.DefaultRetryBackoffType, out.Work.BackoffType,
		)
	})

	t.Run("preserves explicit values", func(t *testing.T) {
		cfg := config.NewDefaultConfig()
		cfg.Work.MaxRetries = 5
		cfg.Work.InitBackoff = 2000
		cfg.Work.MaxBackoff = 30000
		cfg.Work.BackoffType = "fixed"

		out := cfg.WithWorkDefaults()

		testify.Equal(t, 5, out.Work.MaxRetries)
		testify.Equal(t, int64(2000), out.Work.InitBackoff)
		testify.Equal(t, int64(30000), out.Work.MaxBackoff)
		testify.Equal(t, "fixed", out.Work.BackoffType)
	})

	t.Run("does not mutate original", func(t *testing.T) {
		cfg := &config.Config{StepTimeout: 1000}
		_ = cfg.WithWorkDefaults()

		testify.Equal(t, 0, cfg.Work.MaxRetries)
		testify.Equal(t, int64(0), cfg.Work.InitBackoff)
	})
}

func TestValidateNegativeTimeout(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.StepTimeout = -1

	err := cfg.Validate()
	testify.Error(t, err)
	testify.ErrorIs(t, err, config.ErrInvalidStepTimeout)
}
