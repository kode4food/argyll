package main_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	refresh "github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-anthropic/internal/refresh"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const (
	pricingPage = "\n## Model pricing\n" +
		"| Model | Base input tokens | 5m cache writes | 1h cache writes |" +
		" Cache hits and refreshes | Output tokens |\n" + `
| --- | --- | --- | --- | --- | --- |
| Claude Haiku 5.5 | $0.10 / MTok | $0.125 / MTok | $0.20 / MTok | $0.01 / MTok | $0.50 / MTok |
` + "| Claude Haiku 5.5 (for prompts over 100,000 tokens) | $0.50 / MTok | $0.625 / MTok |" +
		" $1 / MTok | $0.05 / MTok | $2.50 / MTok |\n"
	cachingPage = `
  - claude-haiku-5-5-20261119

On the Claude API, the minimum cacheable prompt length is:

* 512 tokens for Claude Haiku 5.5
`
)

var (
	errMissingFixture = errors.New("missing fixture")
)

func TestRefresh(t *testing.T) {
	dir := t.TempDir()
	pricePath := filepath.Join(dir, "model-prices.json")
	thresholdPath := filepath.Join(dir, "prefix-thresholds.json")
	assert.NoError(t, os.WriteFile(pricePath, []byte("{}"), 0o600))
	defaults := []byte(`{"claude-unlisted":1024}`)
	assert.NoError(t, os.WriteFile(thresholdPath, defaults, 0o600))

	err := refresh.Refresh(dir, fetchPages(map[string]string{
		"pricing.md":        pricingPage,
		"prompt-caching.md": cachingPage,
	}))
	assert.NoError(t, err)
	prices, err := catalog.ReadTable[worker.Price](pricePath)
	assert.NoError(t, err)
	price := prices["anthropic/claude-haiku-5-5"]
	assert.Equal(t, new(float64(0.125)), price.CacheOutput)
	if assert.Len(t, price.Tiers, 1) {
		assert.Equal(t, int64(100000), price.Tiers[0].InputOver)
		assert.Equal(t, new(float64(0.625)), price.Tiers[0].CacheOutput)
	}
	assert.Equal(t, price, prices["anthropic/claude-haiku-5-5-20261119"])
	thresholds, err := catalog.ReadTable[int](thresholdPath)
	assert.NoError(t, err)
	assert.Equal(t, 512, thresholds["claude-haiku-5-5"])
	assert.Equal(t, 512, thresholds["claude-haiku-5-5-20261119"])
	assert.Equal(t, 1024, thresholds["claude-unlisted"])
}

func TestInvalidSource(t *testing.T) {
	dir := t.TempDir()
	pricePath := filepath.Join(dir, "model-prices.json")
	assert.NoError(t, os.WriteFile(pricePath, []byte("{}"), 0o600))
	for _, tc := range []struct {
		pages map[string]string
		want  error
	}{
		{
			pages: map[string]string{},
			want:  errMissingFixture,
		},
		{
			pages: map[string]string{"pricing.md": pricingPage},
			want:  errMissingFixture,
		},
		{
			pages: map[string]string{"pricing.md": "# Pricing", "prompt-caching.md": cachingPage},
			want:  refresh.ErrIncompletePricing,
		},
		{
			pages: map[string]string{
				"pricing.md":        strings.Replace(pricingPage, "$0.125 / MTok", "unknown", 1),
				"prompt-caching.md": cachingPage,
			},
			want: refresh.ErrIncompletePricing,
		},
		{
			pages: map[string]string{
				"pricing.md":        strings.Replace(pricingPage, "$0.125", "-", 1),
				"prompt-caching.md": cachingPage,
			},
			want: refresh.ErrIncompletePricing,
		},
		{
			pages: map[string]string{"pricing.md": pricingPage, "prompt-caching.md": "# Caching"},
			want:  refresh.ErrIncompleteThresholds,
		},
		{
			pages: map[string]string{"pricing.md": pricingPage, "prompt-caching.md": cachingPage},
			want:  os.ErrNotExist,
		},
	} {
		assert.ErrorIs(t, refresh.Refresh(dir, fetchPages(tc.pages)), tc.want)
	}
	data, err := os.ReadFile(pricePath)
	assert.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func fetchPages(pages map[string]string) func(string) (string, error) {
	return func(url string) (string, error) {
		if page, ok := pages[filepath.Base(url)]; ok {
			return page, nil
		}
		return "", errMissingFixture
	}
}
