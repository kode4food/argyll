package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-bedrock/internal/bedrock"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestConfig(t *testing.T) {
	credentials(t, true)
	for _, key := range []string{
		"MODEL_CAPABILITIES", "MODEL_PRICES", "MCP_SERVERS", "ARGYLL_MCP_URL", "PROVIDER_URLS",
	} {
		t.Setenv(key, "")
	}
	cfg, p, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{bedrock.Name}, p.Names())
	key := worker.ModelKey{
		Provider: bedrock.Name,
		Model:    "global.anthropic.claude-haiku-5-5",
	}
	assert.Equal(t,
		[]api.Effort{"low", "medium", "high", "xhigh", "max"},
		cfg.Capabilities[key].Efforts)
	if assert.Len(t, cfg.Prices[key].Tiers, 1) {
		assert.Equal(t, int64(100_000), cfg.Prices[key].Tiers[0].InputOver)
	}
	assert.NotNil(t, cfg.Prices[worker.ModelKey{
		Provider: bedrock.Name,
		Model:    "amazon.nova-2-lite-v1:0",
	}].Input)

	t.Setenv("PROVIDER_URLS", "invalid")
	_, _, err = bedrock.LoadConfig()
	assert.ErrorIs(t, err, worker.ErrInvalidProviders)
	t.Setenv("PROVIDER_URLS", "")
	capabilities(t, "{")
	_, _, err = bedrock.LoadConfig()
	assert.Error(t, err)
}
