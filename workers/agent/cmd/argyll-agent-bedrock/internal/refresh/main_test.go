package main_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	refresh "github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-bedrock/internal/refresh"
	"github.com/kode4food/argyll/workers/agent/internal/catalog"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type offerRow struct {
	model        string
	feature      string
	kind         string
	usage        string
	unit         string
	usd          string
	tokenType    string
	cacheFeature string
}

const (
	yes      = "![supported](icon-yes.png)"
	no       = "![not-supported](icon-no.png)"
	textOnly = "| **Input Modalities** | **Output Modalities** |\n" +
		"| --- | --- |\n| " + yes + " Image | " + no + " Image |\n" +
		"| " + yes + " Text | " + yes + " Text |\n\n"

	modelCards = `| Provider | Supported models |
| --- | --- |
| Amazon | [Nova Pro](model-card-amazon-nova-pro.md) |
| Anthropic | [Claude Opus 5](model-card-anthropic-claude-opus-5.md), ` +
		"[Retired](model-card-anthropic-retired.md) |\n" +
		"| Cohere | [Embed v4](model-card-cohere-embed-v4.md) |\n"

	opusCard = textOnly + "| **Messages** | **Responses** | **Converse** | **Invoke** |\n" +
		"| --- | --- | --- | --- |\n" +
		"| " + yes + " | " + no + " | " + yes + " | " + yes + " |\n" +
		"| **Endpoint** | **Model ID** | **In-Region endpoint URL** |" +
		" **Geo inference ID** | **Global inference ID** |\n" +
		"| bedrock-runtime | anthropic.claude-opus-5 | N/A |" +
		" `us.anthropic.claude-opus-5`<br />`eu.anthropic.claude-opus-5` |" +
		" global.anthropic.claude-opus-5 |\n" +
		"| bedrock-mantle | anthropic.claude-opus-5 | https://example | N/A | N/A |\n" +
		"+ **Reasoning:** Supported (effort level configurable — " +
		"low, medium, high; default: medium)\n" +
		"![supported](icon-yes.png) [Client-side tool calling](tool-use.html)\n" +
		"Prompts that exceed 100,000 tokens are billed at higher rates.\n"

	novaCard = textOnly + "| " + yes + " Text | " + yes + " Text | " + yes + " Converse | |\n" +
		"![supported](icon-yes.png) [Client-side tool calling](tool-use.html)\n" +
		"| bedrock-runtime | amazon.nova-pro-v1:0 | https://example |" +
		" Not supported | Not supported |\n"

	embedCard = "| " + yes + " Text | " + no + " Text | " + no + " Converse | |\n" +
		"| bedrock-runtime | cohere.embed-v4:0 | https://example |" +
		" `us.cohere.embed-v4:0` | Not supported |\n"
)

var (
	errOffline = errors.New("offline")
)

func TestPublishedVariants(t *testing.T) {
	for _, tc := range []struct {
		name    string
		card    string
		efforts []api.Effort
		cutoff  int64
	}{
		{
			name: "adaptive effort table",
			card: strings.ReplaceAll(opusCard,
				"effort level configurable — low, medium, high; default: medium",
				"adaptive thinking; effort capped at high when disabled") +
				"[Adaptive thinking](claude-messages-adaptive-thinking.html)",
			efforts: []api.Effort{"low", "medium", "high", "max"},
			cutoff:  100_000,
		},
		{
			name: "K token threshold",
			card: strings.ReplaceAll(opusCard, "Prompts that exceed 100,000 tokens",
				"more than 272K input tokens"),
			efforts: []api.Effort{"low", "medium", "high"},
			cutoff:  272_000,
		},
		{
			name: "reasoning without effort",
			card: strings.ReplaceAll(opusCard,
				" (effort level configurable — low, medium, high; default: medium)", ""),
			cutoff: 100_000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			docs := publishedDocs(t, tc.card)
			err := refresh.Refresh(refresh.RefreshArgs{
				Dir:    dir,
				Region: "us-east-1",
				Fetch:  fixtureFetch(docs),
			})
			if !assert.NoError(t, err) {
				return
			}
			caps, err := catalog.ReadTable[worker.Capabilities](
				filepath.Join(dir, "model-capabilities.json"),
			)
			assert.NoError(t, err)
			assert.Equal(t, tc.efforts, caps["bedrock/anthropic.claude-opus-5"].Efforts)
			prices, err := catalog.ReadTable[worker.Price](filepath.Join(dir, "model-prices.json"))
			assert.NoError(t, err)
			if assert.Len(t, prices["bedrock/anthropic.claude-opus-5"].Tiers, 1) {
				assert.Equal(t,
					tc.cutoff, prices["bedrock/anthropic.claude-opus-5"].Tiers[0].InputOver)
			}
		})
	}
}

func TestIncompleteRefresh(t *testing.T) {
	for _, scenario := range []string{
		"empty catalog", "missing IDs", "missing threshold", "overflow threshold",
		"card error", "adaptive error", "bad capabilities", "bad prices",
	} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			docs := publishedDocs(t, opusCard)
			switch scenario {
			case "empty catalog":
				docs["model-cards.md"] = "# Models"
			case "missing IDs":
				docs["model-card-opus.md"] = "# Model"
			case "missing threshold":
				docs["model-card-opus.md"] = strings.ReplaceAll(opusCard,
					"Prompts that exceed 100,000 tokens", "No published threshold")
			case "overflow threshold":
				docs["model-card-opus.md"] = strings.ReplaceAll(opusCard,
					"100,000", "999999999999999999999999999999")
			case "card error":
				delete(docs, "model-card-opus.md")
			case "adaptive error":
				docs["model-card-opus.md"] = strings.ReplaceAll(
					opusCard,
					"effort level configurable — low, medium, high; default: medium",
					"adaptive",
				) + "[Adaptive thinking](claude-messages-adaptive-thinking.html)"
				delete(docs, "claude-messages-adaptive-thinking.md")
			case "bad capabilities":
				assert.NoError(t, os.WriteFile(filepath.Join(dir, "model-capabilities.json"),
					[]byte("null"), 0o600))
			case "bad prices":
				assert.NoError(t, os.WriteFile(filepath.Join(dir, "model-prices.json"),
					[]byte("null"), 0o600))
			}
			assert.Error(t, refresh.Refresh(refresh.RefreshArgs{
				Dir:    dir,
				Region: "us-east-1",
				Fetch:  fixtureFetch(docs),
			}))
		})
	}
}

func TestOutputModalities(t *testing.T) {
	for _, modality := range []string{"Image", "Audio", "Video", "Speech", "Embedding"} {
		t.Run(modality, func(t *testing.T) {
			dir := t.TempDir()
			docs := publishedDocs(t, opusCard)
			docs["model-cards.md"] += "| Other | [Media](model-card-media.md) |\n"
			card := strings.ReplaceAll(opusCard, "anthropic.claude-opus-5", "media.model")
			docs["model-card-media.md"] = strings.ReplaceAll(card, no+" Image", yes+" "+modality)
			assert.NoError(t, refresh.Refresh(refresh.RefreshArgs{
				Dir:    dir,
				Region: "us-east-1",
				Fetch:  fixtureFetch(docs),
			}))
			caps, err := catalog.ReadTable[worker.Capabilities](
				filepath.Join(dir, "model-capabilities.json"),
			)
			assert.NoError(t, err)
			assert.False(t, caps["bedrock/anthropic.claude-opus-5"].Hidden)
			assert.True(t, caps["bedrock/media.model"].Hidden)
			assert.True(t, caps["bedrock/us.media.model"].Hidden)
		})
	}
}

func TestOfferShapes(t *testing.T) {
	for _, invalid := range []string{"", "NaN", "-1", "invalid"} {
		t.Run("invalid rate "+invalid, func(t *testing.T) {
			docs := publishedDocs(t, opusCard)
			docs["AmazonBedrockFoundationModels/current/us-east-1/index.json"] = offer(t,
				claudeRows([]offerRow{{kind: "Input tokens", usd: invalid}}))
			assert.ErrorIs(t, refresh.Refresh(refresh.RefreshArgs{
				Dir:    t.TempDir(),
				Region: "us-east-1",
				Fetch:  fixtureFetch(docs),
			}), refresh.ErrNoPrices)
		})
	}
	docs := publishedDocs(t, opusCard)
	docs["AmazonBedrockFoundationModels/current/us-east-1/index.json"] = offer(t, []offerRow{
		{
			model:     "Claude Opus 5",
			tokenType: "Input tokens",
			unit:      "1M tokens",
			usd:       "5",
		},
		{
			model:     "Claude Opus 5",
			tokenType: "Output tokens",
			unit:      "1M tokens",
			usd:       "25",
		},
		{
			model:        "Claude Opus 5",
			cacheFeature: "Prompt cache read",
			unit:         "1K tokens",
			usd:          "0.0005",
		},
		{
			model:        "Claude Opus 5",
			cacheFeature: "Prompt cache write",
			unit:         "1K tokens",
			usd:          "0.00625",
		},
		{
			model: "Claude Opus 5",
			kind:  "Input tokens",
			unit:  "requests",
			usd:   "99",
		},
	})
	dir := t.TempDir()
	assert.NoError(t, refresh.Refresh(refresh.RefreshArgs{
		Dir:    dir,
		Region: "us-east-1",
		Fetch:  fixtureFetch(docs),
	}))
	prices, err := catalog.ReadTable[worker.Price](filepath.Join(dir, "model-prices.json"))
	assert.NoError(t, err)
	assert.Equal(t, worker.Price{
		Input:       new(float64(5)),
		Output:      new(float64(25)),
		CacheInput:  new(0.5),
		CacheOutput: new(6.25),
	}, prices["bedrock/anthropic.claude-opus-5"])
}

func TestRefresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model-prices.json")
	retained := `{"bedrock/custom":{"input":1}}`
	if !assert.NoError(t, os.WriteFile(path, []byte(retained), 0o600)) {
		return
	}
	capsPath := filepath.Join(dir, "model-capabilities.json")
	before := []byte(`{
		"bedrock/anthropic.claude-opus-5":{"hidden":true},
		"bedrock/unlisted":{"hidden":true}
	}`)
	if !assert.NoError(t, os.WriteFile(capsPath, before, 0o600)) {
		return
	}
	claude := claudeRows([]offerRow{
		{kind: "Input tokens", usage: "input_tokens", usd: "5.5"},
		{kind: "Input tokens", usage: "input_tokens_global", usd: "5"},
		{kind: "Output tokens", usage: "output_tokens", usd: "27.5"},
		{kind: "Output tokens", usage: "output_tokens_global", usd: "25"},
		{kind: "Prompt cache read input tokens", usage: "cache_read_tokens", usd: "0.55"},
		{kind: "Prompt cache write input tokens", usage: "cache_write_tokens", usd: "6.875"},
		{kind: "Prompt cache write input tokens", usage: "cache_write_tokens_1h", usd: "11"},
		{kind: "Input tokens", usage: "input_tokens_long_ctx", usd: "11"},
		{kind: "Output tokens", usage: "output_tokens_long_ctx", usd: "55"},
	})
	provisioned := offerRow{
		model:   "Claude Opus 5",
		feature: "Provisioned Throughput Inference",
		kind:    "Input tokens",
		usage:   "EUN1-MP:EUN1_Reserved_1Month_InputTPM-Units",
		unit:    "1M tokens",
		usd:     "99",
	}
	nova2 := strings.ReplaceAll(novaCard, "amazon.nova-pro-v1:0", "amazon.nova-2-lite-v1:0")
	nova2 += "+ **Reasoning:** Supported (effort configurable: low, medium, high)\n"
	noTools := strings.ReplaceAll(novaCard, "amazon.nova-pro-v1:0", "deepseek.r1-v1:0")
	noTools = strings.ReplaceAll(noTools,
		"![supported](icon-yes.png) [Client-side tool calling](tool-use.html)\n", "")
	docs := map[string]string{
		"/model-cards.md": modelCards +
			"| Amazon | [Nova 2 Lite](model-card-nova-2-lite.md) |\n" +
			"| DeepSeek | [DeepSeek R1](model-card-deepseek-r1.md) |\n",
		"/model-card-nova-2-lite.md":             nova2,
		"/model-card-deepseek-r1.md":             noTools,
		"/model-card-anthropic-claude-opus-5.md": opusCard,
		"/model-card-amazon-nova-pro.md":         novaCard,
		"/model-card-cohere-embed-v4.md":         embedCard,
		"/AmazonBedrockFoundationModels/current/eu-north-1/index.json": offer(t,
			append(claude, provisioned)),
		"/AmazonBedrock/current/eu-north-1/index.json": offer(t, novaRows([]offerRow{
			{kind: "Input tokens", usage: "input-tokens", usd: "0.00087"},
			{kind: "Output tokens", usage: "output-tokens", usd: "0.00348"},
			{kind: "Input tokens priority", usage: "input-tokens-priority", usd: "0.001"},
			{kind: "input tokens batch", usage: "input-tokens-batch", usd: "0.0004"},
		})),
		"/AmazonBedrockService/current/eu-north-1/index.json": offer(t, []offerRow{
			{
				model: "Nova 2.0 Lite",
				kind:  "Input tokens",
				unit:  "1M tokens",
				usd:   "0.3",
			},
			{
				model: "Nova 2.0 Lite",
				kind:  "Output tokens",
				unit:  "1M tokens",
				usd:   "2.5",
			},
		}),
	}
	fetch := func(url string) (string, error) {
		for suffix, doc := range docs {
			if strings.HasSuffix(url, suffix) {
				return doc, nil
			}
		}
		return "", &catalog.HTTPError{
			URL:    url,
			Status: http.StatusNotFound,
		}
	}

	err := refresh.Refresh(refresh.RefreshArgs{
		Dir:    dir,
		Region: "eu-north-1",
		Fetch:  fetch,
	})
	if !assert.NoError(t, err) {
		return
	}
	prices, err := catalog.ReadTable[worker.Price](path)
	if !assert.NoError(t, err) {
		return
	}
	regional := worker.Price{
		Input:       new(5.5),
		Output:      new(27.5),
		CacheInput:  new(0.55),
		CacheOutput: new(6.875),
		Tiers: []worker.PriceTier{{
			InputOver: 100_000,
			Input:     new(float64(11)),
			Output:    new(float64(55)),
		}},
	}
	assert.Equal(t, map[string]worker.Price{
		"bedrock/custom":                     {Input: new(float64(1))},
		"bedrock/anthropic.claude-opus-5":    regional,
		"bedrock/us.anthropic.claude-opus-5": regional,
		"bedrock/eu.anthropic.claude-opus-5": regional,
		"bedrock/global.anthropic.claude-opus-5": {
			Input:  new(float64(5)),
			Output: new(float64(25)),
		},
		"bedrock/amazon.nova-pro-v1:0": {
			Input:  new(0.87),
			Output: new(3.48),
		},
		"bedrock/amazon.nova-2-lite-v1:0": {
			Input:  new(0.3),
			Output: new(2.5),
		},
	}, prices)
	caps, err := catalog.ReadTable[worker.Capabilities](capsPath)
	assert.NoError(t, err)
	assert.Equal(t, map[string]worker.Capabilities{
		"bedrock/anthropic.claude-opus-5":        {Efforts: []api.Effort{"low", "medium", "high"}},
		"bedrock/us.anthropic.claude-opus-5":     {Efforts: []api.Effort{"low", "medium", "high"}},
		"bedrock/eu.anthropic.claude-opus-5":     {Efforts: []api.Effort{"low", "medium", "high"}},
		"bedrock/global.anthropic.claude-opus-5": {Efforts: []api.Effort{"low", "medium", "high"}},
		"bedrock/amazon.nova-pro-v1:0":           {},
		"bedrock/amazon.nova-2-lite-v1:0":        {Efforts: []api.Effort{"low", "medium", "high"}},
		"bedrock/deepseek.r1-v1:0":               {Hidden: true},
		"bedrock/unlisted":                       {Hidden: true},
		"bedrock/cohere.embed-v4:0":              {Hidden: true},
		"bedrock/us.cohere.embed-v4:0":           {Hidden: true},
	}, caps)
}

func TestRefreshPreservesFiles(t *testing.T) {
	for _, scenario := range []string{"offline", "invalid", "empty", "server error"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			before := []byte(`{"bedrock/kept":{"hidden":true}}`)
			path := filepath.Join(dir, "model-capabilities.json")
			assert.NoError(t, os.WriteFile(path, before, 0o600))
			prices := filepath.Join(dir, "model-prices.json")
			assert.NoError(t, os.WriteFile(prices, []byte(`{}`), 0o600))
			err := refresh.Refresh(refresh.RefreshArgs{
				Dir:    dir,
				Region: "us-east-1",
				Fetch: func(url string) (string, error) {
					if strings.HasSuffix(url, "model-cards.md") {
						return modelCards, nil
					}
					if strings.HasSuffix(url, ".md") {
						return embedCard, nil
					}
					if scenario == "invalid" {
						return "{", nil
					}
					if scenario == "empty" {
						return `{}`, nil
					}
					if scenario == "server error" {
						return "", &catalog.HTTPError{
							URL:    url,
							Status: http.StatusInternalServerError,
						}
					}
					return "", errOffline
				},
			})
			assert.Error(t, err)
			after, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, before, after)
			after, err = os.ReadFile(prices)
			assert.NoError(t, err)
			assert.Equal(t, `{}`, string(after))
		})
	}
}

func TestRefreshFailures(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"model-prices.json", "model-capabilities.json"} {
		path := filepath.Join(dir, name)
		if !assert.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600)) {
			return
		}
	}
	assert.ErrorIs(t, refresh.Refresh(refresh.RefreshArgs{
		Dir:    dir,
		Region: "eu-north-1",
		Fetch: func(string) (string, error) {
			return "", errOffline
		},
	}), errOffline)

	assert.ErrorIs(t, refresh.Refresh(refresh.RefreshArgs{
		Dir:    dir,
		Region: "eu-north-1",
		Fetch: func(url string) (string, error) {
			if strings.HasSuffix(url, "/model-cards.md") {
				return modelCards, nil
			}
			if strings.HasSuffix(url, ".md") {
				return opusCard, nil
			}
			return `{}`, nil
		},
	}), refresh.ErrNoPrices)
}

func claudeRows(rows []offerRow) []offerRow {
	res := make([]offerRow, 0, len(rows))
	for _, r := range rows {
		res = append(res, offerRow{
			model:   "Claude Opus 5",
			feature: "On-demand Inference",
			kind:    r.kind,
			usage:   "EUN1-MP:EUN1_" + r.usage + "-Units",
			unit:    "1M tokens",
			usd:     r.usd,
		})
	}
	return res
}

func novaRows(rows []offerRow) []offerRow {
	res := make([]offerRow, 0, len(rows))
	for _, r := range rows {
		res = append(res, offerRow{
			model:   "Nova Pro",
			feature: "On-demand Inference",
			kind:    r.kind,
			usage:   "EUN1-NovaPro-" + r.usage,
			unit:    "1K tokens",
			usd:     r.usd,
		})
	}
	return res
}

// offer renders rows in the AWS Price List's offer file layout
func offer(t *testing.T, rows []offerRow) string {
	t.Helper()
	products := map[string]any{}
	terms := map[string]any{}
	for i, r := range rows {
		sku := string(rune('A' + i))
		products[sku] = map[string]any{
			"attributes": map[string]string{
				"model":         r.model,
				"feature":       r.feature,
				"inferenceType": r.kind,
				"tokenType":     r.tokenType,
				"featureType":   r.cacheFeature,
				"usagetype":     r.usage,
			},
		}
		terms[sku] = map[string]any{
			sku + ".TERM": map[string]any{
				"priceDimensions": map[string]any{
					sku + ".DIM": map[string]any{
						"unit":         r.unit,
						"pricePerUnit": map[string]string{"USD": r.usd},
					},
				},
			},
		}
	}
	b, err := json.Marshal(map[string]any{
		"products": products,
		"terms":    map[string]any{"OnDemand": terms},
	})
	assert.NoError(t, err)
	return string(b)
}

func fixtureFetch(docs map[string]string) func(string) (string, error) {
	return func(url string) (string, error) {
		for suffix, doc := range docs {
			if strings.HasSuffix(url, "/"+suffix) {
				return doc, nil
			}
		}
		return "", errOffline
	}
}

func publishedDocs(t *testing.T, card string) map[string]string {
	t.Helper()
	prices := offer(t, claudeRows([]offerRow{
		{kind: "Input tokens", usage: "input_tokens", usd: "5"},
		{kind: "Output tokens", usage: "output_tokens", usd: "25"},
		{kind: "Input tokens", usage: "input_tokens_long_ctx", usd: "10"},
		{kind: "Output tokens", usage: "output_tokens_long_ctx", usd: "50"},
	}))
	return map[string]string{
		"model-cards.md":     "| Anthropic | [Claude Opus 5](model-card-opus.md) |\n",
		"model-card-opus.md": card,
		"claude-messages-adaptive-thinking.md": `| Effort level | Thinking behavior |
| --- | --- |
| max | Supported on Claude Opus 5 and Claude Haiku 5.5. |
| xhigh | Supported on Claude Opus 5.5. |
| high (default on most models) | Deep thinking. |
| medium (default on Claude Opus 5.5) | Moderate thinking. |
| low | Minimal thinking. |
`,
		"AmazonBedrock/current/us-east-1/index.json":                 `{}`,
		"AmazonBedrockService/current/us-east-1/index.json":          `{}`,
		"AmazonBedrockFoundationModels/current/us-east-1/index.json": prices,
	}
}
