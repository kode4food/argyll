package archiver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/archiver"
)

func TestLoadFromEnvParsesValues(t *testing.T) {
	t.Setenv("ENGINE_REDIS_ADDR", "engine:6379")
	t.Setenv("FLOW_REDIS_ADDR", "redis:6379")
	t.Setenv("FLOW_REDIS_PASSWORD", "secret")
	t.Setenv("FLOW_REDIS_DB", "2")
	t.Setenv("FLOW_REDIS_PREFIX", "argyll:flow")
	t.Setenv("ARCHIVE_MEMORY_PERCENT", "75.5")
	t.Setenv("ARCHIVE_MAX_AGE", "2h")
	t.Setenv("ARCHIVE_MEMORY_CHECK_INTERVAL", "3s")
	t.Setenv("ARCHIVE_SWEEP_INTERVAL", "30m")
	t.Setenv("ARCHIVE_LEASE_TIMEOUT", "10m")
	t.Setenv("ARCHIVE_PRESSURE_BATCH", "15")
	t.Setenv("ARCHIVE_SWEEP_BATCH", "250")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := archiver.LoadFromEnv()
	assert.NoError(t, err)

	assert.Equal(t, "engine:6379", cfg.EngineStore.Addr)
	assert.Equal(t, "redis:6379", cfg.FlowStore.Addr)
	assert.Equal(t, "secret", cfg.FlowStore.Password)
	assert.Equal(t, 2, cfg.FlowStore.DB)
	assert.Equal(t, "argyll:flow", cfg.FlowStore.Prefix)
	assert.Equal(t, 75.5, cfg.MemoryPercent)
	assert.Equal(t, 2*time.Hour, cfg.MaxAge)
	assert.Equal(t, 3*time.Second, cfg.MemoryCheckInterval)
	assert.Equal(t, 30*time.Minute, cfg.SweepInterval)
	assert.Equal(t, 10*time.Minute, cfg.LeaseTimeout)
	assert.Equal(t, 15, cfg.PressureBatchSize)
	assert.Equal(t, 250, cfg.SweepBatchSize)
	assert.Equal(t, "debug", cfg.LogLevel)
}
