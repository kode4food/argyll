package main_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	refresh "github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-openai/internal/refresh"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const (
	catalogPage = `
## Image
- [Image](/api/docs/models/image.md)
## Realtime & audio
- [Voice](/api/docs/models/voice.md)
## Embedding models
- [Embedding](/api/docs/models/embedding.md)
## ChatGPT models
- [Chat Latest](/api/docs/models/chat-latest.md)
## Flagship models
- [Chat](/api/docs/models/example.md)
- [Long Context](/api/docs/models/long-context.md)
`
	pricingPage = "\n### Standard pricing data\n" +
		"| Model | Short context input | Short context output | Short context cached input |" +
		" Short context 30m write | Short context 12h write | Long context input |" +
		" Long context output | Long context cached input | Long context 30m write |" +
		" Long context 12h write |\n" + `
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| example (<272K context length) | $2 | $10 | $0.1 | $2.5 | $5 | $4 | $15 | $0.2 | $5 | $10 |
| long-context | $3 | $12 | $0.3 | - | - | $6 | $18 | $0.6 | - | - |

| Model | Input | Output | Cached input |
| --- | --- | --- | --- |
| ordinary | $1 | $5 | $0.05 |

### Batch pricing data
| Model | Input | Output | Cached input |
| example | $1 | $5 | $0.05 |

| Category | Model | Input | Output | Cached input | 30m write | 12h write |
| --- | --- | --- | --- | --- | --- | --- |
| ChatGPT | chat-latest | $5 | $30 | $0.50 | - | - |
| Restricted | unavailable (short context) | $10 | $20 | $1 | - | - |
| Restricted | unavailable (long context) | $20 | $30 | $2 | - | - |
`
	modelPage = "## Model details\n- Output modalities: text\n\n" +
		"| Responses | `v1/responses` | Supported |\n" +
		"Reasoning.effort supports: low and high.\n\n## Snapshots\n- `example-2026-01-01`\n"
)

var (
	errMissingFixture = errors.New("missing fixture")
)

func TestRefresh(t *testing.T) {
	path := makeDefaults(t)
	before := []byte(`{"openai/example":{"hidden":true},"openai/legacy-voice":{"hidden":true}}`)
	assert.NoError(t, os.WriteFile(path, before, 0o600))
	err := refresh.Refresh(filepath.Dir(path), fetchPages(map[string]string{
		"all.md":     catalogPage,
		"pricing.md": pricingPage,
		"example.md": modelPage,
		"long-context.md": "\n- Output modalities: text\n" +
			"| Chat Completions | `v1/chat/completions` | Supported |\n" +
			"Prompts with more than 512K input tokens have long-context rates.",
	}))
	assert.NoError(t, err)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	want := `{
		"openai/image": {"hidden":true},
		"openai/voice": {"hidden":true},
		"openai/embedding": {"hidden":true},
		"openai/legacy-voice": {"hidden":true},
		"openai/chat-latest": {"hidden":true},
		"openai/example": {"endpoint":"responses","efforts":["low","high"]},
		"openai/example-2026-01-01": {"endpoint":"responses","efforts":["low","high"]},
		"openai/long-context": {"endpoint":"chat_completions"}
	}`
	assert.JSONEq(t, want, string(data))
	prices, err := catalog.ReadTable[worker.Price](
		filepath.Join(filepath.Dir(path), "model-prices.json"),
	)
	assert.NoError(t, err)
	rate := prices["openai/example"]
	assert.Equal(t, new(float64(2)), rate.Input)
	assert.Equal(t, new(float64(10)), rate.Output)
	assert.Equal(t, new(float64(0.1)), rate.CacheInput)
	assert.Equal(t, new(float64(2.5)), rate.CacheOutput)
	if assert.Len(t, rate.Tiers, 1) {
		assert.Equal(t, int64(272000), rate.Tiers[0].InputOver)
	}
	long := prices["openai/long-context"]
	if assert.Len(t, long.Tiers, 1) {
		assert.Equal(t, int64(512000), long.Tiers[0].InputOver)
	}
	assert.NotContains(t, prices, "openai/unavailable")
	assert.Equal(t, rate, prices["openai/example-2026-01-01"])
}

func TestOutputModalities(t *testing.T) {
	for _, modality := range []string{"audio", "image", "video", "text, audio", "text, image"} {
		t.Run(modality, func(t *testing.T) {
			path := makeDefaults(t)
			assert.NoError(t, os.WriteFile(path, []byte(`{"openai/example":{}}`), 0o600))
			doc := strings.ReplaceAll(modelPage, "Output modalities: text",
				"Output modalities: "+modality)
			err := refresh.Refresh(filepath.Dir(path), fetchPages(map[string]string{
				"all.md":     catalogPage,
				"pricing.md": pricingPage,
				"example.md": doc,
			}))
			assert.NoError(t, err)
			models, err := catalog.ReadTable[worker.Capabilities](path)
			assert.NoError(t, err)
			assert.True(t, models["openai/example"].Hidden)
			assert.True(t, models["openai/example-2026-01-01"].Hidden)
		})
	}
}

func TestInvalidSource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		catalog string
		pricing string
		want    error
	}{
		{
			name:    "missing category",
			catalog: "# Catalog",
			pricing: pricingPage,
			want:    refresh.ErrIncompleteCatalog,
		},
		{
			name:    "invalid price",
			catalog: catalogPage,
			pricing: strings.Replace(pricingPage, "$2", "?", 1),
			want:    catalog.ErrInvalidRate,
		},
		{
			name:    "missing table",
			catalog: catalogPage,
			pricing: "# Pricing",
			want:    refresh.ErrIncompletePricing,
		},
		{
			name:    "missing specialized table",
			catalog: catalogPage,
			pricing: strings.Split(pricingPage, "| Category")[0],
			want:    refresh.ErrIncompletePricing,
		},
		{
			name:    "invalid long-context price",
			catalog: catalogPage,
			pricing: strings.Replace(pricingPage, "$15", "?", 1),
			want:    catalog.ErrInvalidRate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := makeDefaults(t)
			err := refresh.Refresh(filepath.Dir(path), fetchPages(map[string]string{
				"all.md":     tc.catalog,
				"pricing.md": tc.pricing,
			}))
			assert.ErrorIs(t, err, tc.want)
			data, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, "{}", string(data))
		})
	}
}

func TestFetchFailure(t *testing.T) {
	path := makeDefaults(t)
	assert.ErrorIs(t, refresh.Refresh(filepath.Dir(path), fetchPages(nil)), errMissingFixture)
	err := refresh.Refresh(filepath.Dir(path), fetchPages(map[string]string{"all.md": catalogPage}))
	assert.ErrorIs(t, err, errMissingFixture)
}

func makeDefaults(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model-capabilities.json")
	assert.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "model-prices.json"), []byte("{}"), 0o600))
	return path
}

func fetchPages(pages map[string]string) func(string) (string, error) {
	return func(url string) (string, error) {
		if page, ok := pages[filepath.Base(url)]; ok {
			return page, nil
		}
		return "", errMissingFixture
	}
}
