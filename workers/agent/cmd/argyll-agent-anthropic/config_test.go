package main_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/argyll/workers/agent/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	for _, key := range []string{
		"MODEL_CAPABILITIES", "MODEL_PRICES", "MCP_SERVERS", "ARGYLL_MCP_URL",
		"PROVIDER_URLS", "PREFIX_THRESHOLDS",
	} {
		t.Setenv(key, "")
	}
	cfg, _, err := anthropic.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.NotEmpty(t, cfg.Prices)
	t.Setenv("PROVIDER_URLS", "invalid")
	_, _, err = anthropic.LoadConfig()
	assert.ErrorIs(t, err, worker.ErrInvalidProviders)
	t.Setenv("PROVIDER_URLS", "")
	path := filepath.Join(t.TempDir(), "thresholds.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"bad":512}`), 0o600))
	t.Setenv("PREFIX_THRESHOLDS", path)
	_, _, err = anthropic.LoadConfig()
	assert.ErrorIs(t, err, worker.ErrInvalidModelKey)
}
