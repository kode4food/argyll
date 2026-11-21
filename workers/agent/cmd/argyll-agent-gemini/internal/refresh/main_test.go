package main_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	refresh "github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-gemini/internal/refresh"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const (
	docsURL = "https://ai.google.dev/gemini-api/docs/"

	modelsDoc = `# Models

## Gemini 4  {% id="gemini-4" %}

  * ` + "`gemini-4-neon`" + `

## Audio models  {% id="audio-models" %}

{% table variant="model" %}
  * Model
  * Endpoint

  ---

  * [Gemini TTS](/docs/models/gemini-tts)
  * ` + "`gemini-tts`" + `
{% /table %}

## Generative media models  {% id="generative-media-models" %}

  * ` + "`gemini-image`" + `

## Gemini Old  {% id="gemini-old" %}

  ` + "`gemini-old`" + ` is [deprecated](/docs/deprecations#old) and routes to
  [` + "`gemini-4-neon`" + `](/docs/models/gemini-4-neon).

  * ` + "`gemini-old`" + `

## Gemini Lite  {% id="gemini-lite" %}

  ` + "`gemini-lite`" + ` is not deprecated.

  * ` + "`gemini-lite`" + `
`

	thinkingDoc = `## Controlling thinking  {% id="thinking-levels" %}

| Model | Default Thinking | Levels Supported |
| --- | --- | --- |
| gemini-4-neon | On (high) | low, medium, high, xhigh |
| gemini-pro | On (high) | low, high |
| gemini-image | On (minimal) | minimal, high |
`

	pricingDoc = `# Gemini Developer API pricing

## Gemini 4 Neon  {% id="gemini-4-neon" %}

{% two-column-layout %}
  {% left-column class="center-vertically" %}
  *[` + "`gemini-4-neon`" + `](/docs/models/gemini-4-neon)*
  {% /left-column %}
{% /two-column-layout %}

{% hr /%}

{% tabs %}
  {% tab title="Standard" %}
  {% table %}
  * Feature
  * Free Tier
  * Paid Tier, per 1M tokens in USD

  ---

  * Input price
  * Not available
  * $2.00 through November 30, 2026.
  $4.00 starting December 1, 2026.

  ---

  * Output price (including thinking tokens)
  * Not available
  * $10.00 through November 30, 2026.
  $20.00 starting December 1, 2026.

  ---

  * Context caching price
  * Not available
  * $0.10 through November 30, 2026.
  $0.20 starting December 1, 2026.
  $1.00 / 1,000,000 tokens per hour (storage price)
  {% /table %}
  {% /tab %}

  {% tab title="Batch" %}
  {% table %}
  * Feature
  * Free Tier
  * Paid Tier, per 1M tokens in USD

  ---

  * Input price
  * Not available
  * $1.00
  {% /table %}
  {% /tab %}
{% /tabs %}

## Gemini Pro  {% id="gemini-pro" %}

*[` + "`gemini-pro`" + `](/docs/models/gemini-pro) and
  [` + "`gemini-pro-tools`" + `](/docs/models/gemini-pro#tools)*

{% hr /%}

{% table %}
  * Feature
  * Free Tier
  * Paid Tier, per 1M tokens in USD

  ---

  * Input price
  * Free of charge
  * $2.00 (text / image / video), prompts <= 200k tokens
  $3.00 (audio)
  $4.00, prompts > 200k tokens

  ---

  * Output price (including thinking tokens)
  * Free of charge
  * $12.00, prompts <= 200k tokens
  $18.00, prompts > 200k

  ---

  * Context caching price
  * Not available
  * $0.20
{% /table %}

## Gemini Image  {% id="gemini-image" %}

*[` + "`gemini-image`" + `](/docs/models/gemini-image)*

{% hr /%}

{% table %}
  * Input price
  * Free of charge
  * $1.00
{% /table %}

## Gemini Decision  {% id="gemini-decision" %}

*[` + "`gemini-decision`" + `](/docs/models/gemini-decision)*

{% hr /%}

{% table %}
  * Input price
  * Not available
  * $1.00
{% /table %}
`
)

var (
	errOffline = errors.New("offline")
)

func TestRefresh(t *testing.T) {
	for _, tc := range []struct {
		name  string
		today time.Time
		input float64
		cache float64
	}{
		{
			name:  "rates through a date",
			today: time.Date(2026, time.November, 30, 23, 0, 0, 0, time.UTC),
			input: 2,
			cache: 0.1,
		},
		{
			name:  "rates starting a date",
			today: time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
			input: 4,
			cache: 0.2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			err := refresh.Refresh(refresh.RefreshArgs{
				Dir:   dir,
				Fetch: fixtureFetch(publishedDocs()),
				Today: tc.today,
			})
			if !assert.NoError(t, err) {
				return
			}
			caps, err := catalog.ReadTable[worker.Capabilities](
				filepath.Join(dir, "model-capabilities.json"),
			)
			assert.NoError(t, err)
			assert.Equal(t, map[string]worker.Capabilities{
				"gemini/gemini-4-neon": {Efforts: []api.Effort{"low", "medium", "high", "xhigh"}},
				"gemini/gemini-pro":    {Efforts: []api.Effort{"low", "high"}},
				"gemini/gemini-image":  {Hidden: true},
				"gemini/gemini-old":    {Hidden: true},
				"gemini/gemini-tts":    {Hidden: true},
			}, caps)
			prices, err := catalog.ReadTable[worker.Price](filepath.Join(dir, "model-prices.json"))
			assert.NoError(t, err)
			neon := prices["gemini/gemini-4-neon"]
			assert.Equal(t, tc.input, *neon.Input)
			assert.Equal(t, tc.input*5, *neon.Output)
			assert.Equal(t, tc.cache, *neon.CacheInput)
			assert.Nil(t, neon.CacheOutput)
			assert.Empty(t, neon.Tiers)

			pro := worker.Price{
				Input:      new(float64(2)),
				Output:     new(float64(12)),
				CacheInput: new(0.2),
				Tiers: []worker.PriceTier{{
					InputOver:  200_000,
					Input:      new(float64(4)),
					Output:     new(float64(18)),
					CacheInput: new(0.2),
				}},
			}
			assert.Equal(t, pro, prices["gemini/gemini-pro"])
			assert.Equal(t, pro, prices["gemini/gemini-pro-tools"])
			assert.NotContains(t, prices, "gemini/gemini-image")
			assert.NotContains(t, prices, "gemini/gemini-decision")
		})
	}
}

func TestRefreshFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		docs map[string]string
		err  error
	}{
		{
			name: "unreachable docs",
			docs: map[string]string{},
			err:  errOffline,
		},
		{
			name: "no thinking table",
			docs: withDoc("thinking", "# Gemini thinking\n"),
			err:  refresh.ErrIncompleteModels,
		},
		{
			name: "no media models",
			docs: withDoc("models", "# Models\n"),
			err:  refresh.ErrIncompleteModels,
		},
		{
			name: "no prices",
			docs: withDoc("pricing", "# Pricing\n"),
			err:  refresh.ErrNoPrices,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			err := refresh.Refresh(refresh.RefreshArgs{
				Dir:   dir,
				Fetch: fixtureFetch(tc.docs),
				Today: time.Now(),
			})
			assert.ErrorIs(t, err, tc.err)
			caps, err := catalog.ReadTable[worker.Capabilities](
				filepath.Join(dir, "model-capabilities.json"),
			)
			assert.NoError(t, err)
			assert.Empty(t, caps)
		})
	}
}

func publishedDocs() map[string]string {
	return map[string]string{
		docsURL + "models":   modelsDoc,
		docsURL + "thinking": thinkingDoc,
		docsURL + "pricing":  pricingDoc,
	}
}

func withDoc(page, doc string) map[string]string {
	docs := publishedDocs()
	docs[docsURL+page] = doc
	return docs
}

func fixtureFetch(docs map[string]string) func(string) (string, error) {
	return func(url string) (string, error) {
		if doc, ok := docs[url]; ok {
			return doc, nil
		}
		return "", errOffline
	}
}
