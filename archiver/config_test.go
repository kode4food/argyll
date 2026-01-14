package archiver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/archiver"
)

func TestLoadFromEnvRequiresBucketURL(t *testing.T) {
	t.Setenv("FLOW_REDIS_ADDR", "localhost:6379")
	t.Setenv("ARCHIVE_BUCKET_URL", "")

	_, err := archiver.LoadFromEnv()
	assert.ErrorIs(t, err, archiver.ErrBucketURLRequired)
}

func TestLoadFromEnvParsesValues(t *testing.T) {
	t.Setenv("FLOW_REDIS_ADDR", "redis:6379")
	t.Setenv("FLOW_REDIS_PASSWORD", "secret")
	t.Setenv("FLOW_REDIS_DB", "2")
	t.Setenv("FLOW_REDIS_PREFIX", "argyll:flow")
	t.Setenv("ARCHIVE_BUCKET_URL", "mem://archiver-test")
	t.Setenv("ARCHIVE_PREFIX", "archived/")
	t.Setenv("ARCHIVE_POLL_INTERVAL", "250ms")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := archiver.LoadFromEnv()
	assert.NoError(t, err)

	assert.Equal(t, "redis:6379", cfg.FlowStore.Addr)
	assert.Equal(t, "secret", cfg.FlowStore.Password)
	assert.Equal(t, 2, cfg.FlowStore.DB)
	assert.Equal(t, "argyll:flow", cfg.FlowStore.Prefix)
	assert.Equal(t, "mem://archiver-test", cfg.BucketURL)
	assert.Equal(t, "archived/", cfg.Prefix)
	assert.Equal(t, 250*time.Millisecond, cfg.PollInterval)
	assert.Equal(t, "debug", cfg.LogLevel)
}
