package archive_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/archive"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PARTITION_REDIS_ADDR", "redis:6379")
	t.Setenv("PARTITION_REDIS_PASSWORD", "secret")
	t.Setenv("PARTITION_REDIS_DB", "2")
	t.Setenv("PARTITION_REDIS_PREFIX", "argyll:flow")
	t.Setenv("ARCHIVE_MEMORY_PERCENT", "75.5")
	t.Setenv("ARCHIVE_MAX_AGE", "2h")
	t.Setenv("ARCHIVE_MEMORY_CHECK_INTERVAL", "3s")
	t.Setenv("ARCHIVE_POLL_INTERVAL", "250ms")
	t.Setenv("ARCHIVE_SWEEP_INTERVAL", "30m")
	t.Setenv("ARCHIVE_LEASE_TIMEOUT", "10m")
	t.Setenv("ARCHIVE_PRESSURE_BATCH", "15")
	t.Setenv("ARCHIVE_SWEEP_BATCH", "250")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := archive.LoadFromEnv()
	assert.NoError(t, err)

	assert.Equal(t, "redis:6379", cfg.FlowStore.Addr)
	assert.Equal(t, "secret", cfg.FlowStore.Password)
	assert.Equal(t, 2, cfg.FlowStore.DB)
	assert.Equal(t, "argyll:flow", cfg.FlowStore.Prefix)
	assert.Equal(t, 75.5, cfg.MemoryPercent)
	assert.Equal(t, 2*time.Hour, cfg.MaxAge)
	assert.Equal(t, 3*time.Second, cfg.MemoryCheckInterval)
	assert.Equal(t, 250*time.Millisecond, cfg.PollInterval)
	assert.Equal(t, 30*time.Minute, cfg.SweepInterval)
	assert.Equal(t, 10*time.Minute, cfg.LeaseTimeout)
	assert.Equal(t, 15, cfg.PressureBatchSize)
	assert.Equal(t, 250, cfg.SweepBatchSize)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestDefaultPrefix(t *testing.T) {
	t.Setenv("PARTITION_REDIS_ADDR", "")
	t.Setenv("PARTITION_REDIS_PASSWORD", "")
	t.Setenv("PARTITION_REDIS_DB", "")
	t.Setenv("PARTITION_REDIS_PREFIX", "")
	t.Setenv("PARTITION_SNAPSHOT_WORKERS", "")
	t.Setenv("ARCHIVE_MEMORY_PERCENT", "")
	t.Setenv("ARCHIVE_MAX_AGE", "")
	t.Setenv("ARCHIVE_MEMORY_CHECK_INTERVAL", "")
	t.Setenv("ARCHIVE_POLL_INTERVAL", "")
	t.Setenv("ARCHIVE_SWEEP_INTERVAL", "")
	t.Setenv("ARCHIVE_LEASE_TIMEOUT", "")
	t.Setenv("ARCHIVE_PRESSURE_BATCH", "")
	t.Setenv("ARCHIVE_SWEEP_BATCH", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := archive.LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, archive.DefaultStorePrefix, cfg.FlowStore.Prefix)
	assert.Equal(t, archive.DefaultPollInterval, cfg.PollInterval)
}

func TestConfigValidate(t *testing.T) {
	cfg := archive.Config{
		MemoryCheckInterval: time.Second,
		PollInterval:        time.Second,
		SweepInterval:       time.Second,
		LeaseTimeout:        time.Second,
		PressureBatchSize:   1,
		SweepBatchSize:      1,
	}

	tests := []struct {
		name string
		mut  func(*archive.Config)
		want error
	}{
		{
			name: "memory check interval",
			mut: func(cfg *archive.Config) {
				cfg.MemoryCheckInterval = 0
			},
			want: archive.ErrMemoryCheckIntervalInvalid,
		},
		{
			name: "poll interval",
			mut: func(cfg *archive.Config) {
				cfg.PollInterval = 0
			},
			want: archive.ErrPollIntervalInvalid,
		},
		{
			name: "sweep interval",
			mut: func(cfg *archive.Config) {
				cfg.SweepInterval = 0
			},
			want: archive.ErrSweepIntervalInvalid,
		},
		{
			name: "lease timeout",
			mut: func(cfg *archive.Config) {
				cfg.LeaseTimeout = 0
			},
			want: archive.ErrLeaseTimeoutInvalid,
		},
		{
			name: "pressure batch size",
			mut: func(cfg *archive.Config) {
				cfg.PressureBatchSize = 0
			},
			want: archive.ErrPressureBatchInvalid,
		},
		{
			name: "sweep batch size",
			mut: func(cfg *archive.Config) {
				cfg.SweepBatchSize = 0
			},
			want: archive.ErrSweepBatchInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curr := cfg
			tt.mut(&curr)
			err := curr.Validate()
			assert.ErrorIs(t, err, tt.want)
		})
	}
}
