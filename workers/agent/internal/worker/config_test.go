package worker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const crmToken = "secret"

func TestLoadConfig(t *testing.T) {
	t.Setenv("ARGYLL_MCP_URL", "")
	t.Run("reads servers, prices, and providers", func(t *testing.T) {
		t.Setenv("CRM_TOKEN", crmToken)
		t.Setenv("MCP_SERVERS", writeFile(t, `{"crm": {"url": "http://crm/mcp",
			"headers": {"Authorization": "Bearer ${CRM_TOKEN}"}}}`))
		prices := writeFile(t, `{
			"anthropic/claude-sonnet-5-5": {
				"input": 3, "output": 15, "cache_input": 0.3, "cache_output": 3.75
			},
			"groq/openai/gpt-oss-20b": {
				"input": 0.075, "tiers": [{"input_over": 128000, "input": 0.15}]
			}
		}`)
		t.Setenv("MODEL_PRICES", prices)
		t.Setenv("PROVIDER_URLS", "groq=https://api.groq.com/openai/v1, ollama=http://ollama")

		cfg, err := worker.LoadConfig(worker.Defaults{
			Prices: `{
				"anthropic/claude-sonnet-5-5": {"input": 2},
				"anthropic/claude-haiku-5-5": {"input": 0.1, "output": 0.5}
			}`,
			Capabilities: `{
				"anthropic/claude-haiku-5-5": {"efforts": ["low"]},
				"groq/whisper": {"hidden": true}
			}`,
		})
		assert.NoError(t, err)
		want := &worker.Config{
			Capabilities: map[worker.ModelKey]worker.Capabilities{
				{
					Provider: "anthropic",
					Model:    "claude-haiku-5-5",
				}: {Efforts: []api.Effort{"low"}},
			},
			Servers: map[string]worker.MCPServer{
				"crm": {
					URL:     "http://crm/mcp",
					Headers: map[string]string{"Authorization": "Bearer " + crmToken},
				},
			},
			Providers: map[api.ProviderID]string{
				"groq":   "https://api.groq.com/openai/v1",
				"ollama": "http://ollama",
			},
			Prices: map[worker.ModelKey]worker.Price{
				{
					Provider: "anthropic",
					Model:    "claude-sonnet-5-5",
				}: {
					Input:       new(float64(3)),
					Output:      new(float64(15)),
					CacheInput:  new(0.3),
					CacheOutput: new(3.75),
				},
				{
					Provider: "anthropic",
					Model:    "claude-haiku-5-5",
				}: {
					Input:  new(0.1),
					Output: new(0.5),
				},
				{
					Provider: "groq",
					Model:    "openai/gpt-oss-20b",
				}: {
					Input: new(0.075),
					Tiers: []worker.PriceTier{{
						InputOver: 128000,
						Input:     new(0.15),
					}},
				},
			},
		}
		assert.Equal(t, want, cfg)
	})

	t.Run("rejects invalid configuration", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			servers   string
			providers string
			want      error
		}{
			{
				name:    "bad server name",
				servers: `{"bad name": {"url": "http://x"}}`,
				want:    worker.ErrInvalidServerName,
			},
			{
				name:    "missing url",
				servers: `{"crm": {}}`,
				want:    worker.ErrMissingServerURL,
			},
			{
				name:    "missing env",
				servers: `{"crm": {"url": "http://x", "headers": {"A": "${NOPE_UNSET}"}}}`,
				want:    worker.ErrMissingEnv,
			},
			{
				name:      "bad providers",
				servers:   `{}`,
				providers: "groq",
				want:      worker.ErrInvalidProviders,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv("MCP_SERVERS", writeFile(t, tc.servers))
				t.Setenv("MODEL_PRICES", "")
				t.Setenv("PROVIDER_URLS", tc.providers)
				_, err := worker.LoadConfig(worker.Defaults{Prices: `{}`})
				assert.ErrorIs(t, err, tc.want)
			})
		}
	})

	t.Run("price file cannot be read", func(t *testing.T) {
		t.Setenv("MCP_SERVERS", "")
		t.Setenv("MODEL_PRICES", filepath.Join(t.TempDir(), "missing.json"))
		_, err := worker.LoadConfig(worker.Defaults{Prices: `{}`})
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("rejects a price key without a provider", func(t *testing.T) {
		t.Setenv("MCP_SERVERS", "")
		t.Setenv("MODEL_PRICES", writeFile(t, `{"claude": {"input": 1}}`))
		_, err := worker.LoadConfig(worker.Defaults{Prices: `{}`})
		assert.ErrorIs(t, err, worker.ErrInvalidModelKey)
	})

	t.Run("rejects duplicate price tiers", func(t *testing.T) {
		t.Setenv("MCP_SERVERS", "")
		t.Setenv("MODEL_PRICES", "")
		_, err := worker.LoadConfig(worker.Defaults{
			Prices: `{"local/model":{"tiers":[{"input_over":4},{"input_over":4}]}}`,
		})
		assert.ErrorIs(t, err, worker.ErrInvalidPriceTier)
	})

	t.Run("offers the default Argyll MCP alongside configured servers", func(t *testing.T) {
		t.Setenv("ARGYLL_MCP_URL", "http://argyll-mcp:8071/mcp")
		t.Setenv("MCP_SERVERS", writeFile(t, `{"tools":{"url":"http://tools/mcp"}}`))
		t.Setenv("MODEL_PRICES", "")
		t.Setenv("PROVIDER_URLS", "")
		cfg, err := worker.LoadConfig(worker.Defaults{Prices: `{}`})
		assert.NoError(t, err)
		assert.Equal(t, "http://argyll-mcp:8071/mcp", cfg.Servers["argyll"].URL)
		assert.Equal(t, "http://tools/mcp", cfg.Servers["tools"].URL)
	})
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
