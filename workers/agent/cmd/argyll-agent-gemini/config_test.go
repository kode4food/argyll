package main_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-gemini/internal/gemini"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestConfig(t *testing.T) {
	credentials(t, true)
	for _, key := range []string{
		"MODEL_CAPABILITIES", "MODEL_PRICES", "MCP_SERVERS", "ARGYLL_MCP_URL", "PROVIDER_URLS",
	} {
		t.Setenv(key, "")
	}
	cfg, p, err := gemini.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{gemini.Name}, p.Names())
	pro := worker.ModelKey{
		Provider: gemini.Name,
		Model:    "gemini-3.1-pro-preview",
	}
	assert.Equal(t, []api.Effort{"low", "medium", "high"}, cfg.Capabilities[pro].Efforts)
	if assert.Len(t, cfg.Prices[pro].Tiers, 1) {
		assert.Equal(t, int64(200_000), cfg.Prices[pro].Tiers[0].InputOver)
	}
	assert.True(t, cfg.Capabilities[worker.ModelKey{
		Provider: gemini.Name,
		Model:    "gemini-3.8-flash-tts",
	}].Hidden)

	t.Setenv("PROVIDER_URLS", "invalid")
	_, _, err = gemini.LoadConfig()
	assert.ErrorIs(t, err, worker.ErrInvalidProviders)
	t.Setenv("PROVIDER_URLS", "")
	path := filepath.Join(t.TempDir(), "capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte("{"), 0o600))
	t.Setenv("MODEL_CAPABILITIES", path)
	_, _, err = gemini.LoadConfig()
	assert.Error(t, err)
}
