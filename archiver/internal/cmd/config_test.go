package cmd_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/archiver/internal/cmd"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg := cmd.Config{}
	cmd.LoadConfig(&cfg)

	assert.Equal(t, cmd.DefaultPollInterval, cfg.PollInterval)
}

func TestLoadConfigPreservesExplicitValue(t *testing.T) {
	cfg := cmd.Config{
		PollInterval: 2 * time.Second,
	}
	cmd.LoadConfig(&cfg)

	assert.Equal(t, 2*time.Second, cfg.PollInterval)
}

func TestLoadConfigUsesEnvOverride(t *testing.T) {
	t.Setenv("ARCHIVE_POLL_INTERVAL", "250ms")

	cfg := cmd.Config{}
	cmd.LoadConfig(&cfg)

	assert.Equal(t, 250*time.Millisecond, cfg.PollInterval)
}

func TestLoadConfigNilIsNoop(t *testing.T) {
	assert.NotPanics(t, func() {
		cmd.LoadConfig(nil)
	})
}
