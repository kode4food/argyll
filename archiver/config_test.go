package archiver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/archiver"
)

func TestLoadFromEnvParsesValues(t *testing.T) {
	t.Setenv("PARTITION_REDIS_ADDR", "redis:6379")
	t.Setenv("PARTITION_REDIS_PASSWORD", "secret")
	t.Setenv("PARTITION_REDIS_DB", "2")
	t.Setenv("PARTITION_REDIS_PREFIX", "argyll:partition")
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

	assert.Equal(t, "redis:6379", cfg.PartitionStore.Addr)
	assert.Equal(t, "secret", cfg.PartitionStore.Password)
	assert.Equal(t, 2, cfg.PartitionStore.DB)
	assert.Equal(t, "argyll:partition", cfg.PartitionStore.Prefix)
	assert.Equal(t, 75.5, cfg.MemoryPercent)
	assert.Equal(t, 2*time.Hour, cfg.MaxAge)
	assert.Equal(t, 3*time.Second, cfg.MemoryCheckInterval)
	assert.Equal(t, 30*time.Minute, cfg.SweepInterval)
	assert.Equal(t, 10*time.Minute, cfg.LeaseTimeout)
	assert.Equal(t, 15, cfg.PressureBatchSize)
	assert.Equal(t, 250, cfg.SweepBatchSize)
	assert.Equal(t, "debug", cfg.LogLevel)
}
