package main_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/workers/agent/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestDefaultProvider(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("MODEL_CAPABILITIES", "")
	_, p, err := openai.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{openai.Name}, p.Names())
	assert.True(t, p.Supports(openai.Name))
	assert.False(t, p.Supports("anthropic"))
}

func TestConfiguredProvider(t *testing.T) {
	t.Setenv("MODEL_CAPABILITIES", "")
	t.Setenv("OPENAI_API_KEY", "")
	p, err := openai.NewProvider(&worker.Config{
		Providers: map[api.ProviderID]string{"anthropic": "http://gateway"},
	})
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, p.Supports("anthropic"))
	assert.Equal(t, []api.ProviderID{"anthropic"}, p.Names())
}

func TestNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = io.WriteString(w, `{"choices":[],"usage":{"prompt_tokens":10,
			"completion_tokens":2,"cost":0.125}}`)
	}))
	defer srv.Close()
	t.Setenv("MODEL_CAPABILITIES", "")

	p, err := openai.NewProvider(&worker.Config{
		Providers: map[api.ProviderID]string{"local": srv.URL},
		Capabilities: map[worker.ModelKey]worker.Capabilities{
			{Provider: "local", Model: "m"}: {},
		},
	})
	if !assert.NoError(t, err) {
		return
	}
	c, err := p.Open(&worker.Chat{Config: &api.AgentConfig{
		Provider: "local",
		Model:    "m",
	}})
	if !assert.NoError(t, err) {
		return
	}
	turn, err := c.Turn(t.Context())
	assert.True(t, errors.Is(err, openai.ErrNoChoices))
	assert.Equal(t, int64(10), turn.InputTokens)
	assert.Equal(t, int64(2), turn.OutputTokens)
	assert.Equal(t, new(0.125), turn.Cost)
}

func TestModelEfforts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"gpt-5.4-nano"},
				{"id":"gpt-5.4-nano-2026-03-17"},
				{"id":"gpt-image-2.6-flare"},
				{"id":"gpt-transcribe"},
				{"id":"text-embedding-3-small"},
				{"id":"gpt-realtime-2.1"},
				{"id":"gpt-audio-1.5"},
				{"id":"gpt-audio-mini-2025-12-15"},
				{"id":"tts-1-hd-1106"},
				{"id":"chat-latest"},
				{"id":"new-model"}
			]}`)
		case "/chat/completions":
			var req struct {
				Effort api.Effort `json:"reasoning_effort"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, api.Effort("low"), req.Effort)
			_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	isolateConfig(t)
	t.Setenv("OPENAI_BASE_URL", srv.URL+"/")
	t.Setenv("OPENAI_API_KEY", "test")

	conf, p, err := openai.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	models := listModels(t, conf, p)
	efforts := []api.Effort{"none", "low", "medium", "high", "xhigh"}
	assert.Equal(t, []api.AgentModel{
		{ID: "gpt-5.4-nano", Efforts: efforts},
		{ID: "gpt-5.4-nano-2026-03-17", Efforts: efforts},
		{ID: "new-model"},
	}, models)
	for _, model := range []api.ModelID{"gpt-5.4-nano", "gpt-5.4-nano-2026-03-17"} {
		cfg := &api.AgentConfig{
			Provider: openai.Name,
			Model:    model,
			Effort:   "low",
		}
		c, err := p.Open(&worker.Chat{Config: cfg})
		if !assert.NoError(t, err) {
			return
		}
		_, err = c.Turn(t.Context())
		assert.NoError(t, err)
	}
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{
		Provider: openai.Name,
		Model:    "new-model",
	}})
	assert.NoError(t, err)
	for _, model := range []api.ModelID{
		"gpt-image-2.6-flare", "gpt-audio-1.5", "gpt-realtime-2.1",
	} {
		_, err := p.Open(&worker.Chat{Config: &api.AgentConfig{
			Provider: openai.Name,
			Model:    model,
		}})
		assert.ErrorIs(t, err, openai.ErrUnsupportedModel)
		problem, ok := errors.AsType[*api.ProblemDetails](err)
		if assert.True(t, ok) {
			assert.Equal(t, http.StatusUnprocessableEntity, problem.Status)
		}
	}
	_, err = p.Open(&worker.Chat{Config: &api.AgentConfig{
		Provider: openai.Name,
		Model:    "gpt-5.4-nano",
		Effort:   "unsupported",
	}})
	assert.ErrorIs(t, err, openai.ErrInvalidEffort)
}

func TestCapabilityOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"org/model"},{"id":"org/hidden"}]}`)
		case "/responses":
			var req struct {
				Reasoning struct{ Effort api.Effort }
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, api.Effort("high"), req.Reasoning.Effort)
			_, _ = io.WriteString(w, `{"status":"completed","output":[],"usage":{"cost":0}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "models.json")
	data := []byte(`{
		"local/org/model":{"endpoint":"responses","efforts":["high"]},
		"local/org/hidden":{"hidden":true}
	}`)
	if !assert.NoError(t, os.WriteFile(path, data, 0o600)) {
		return
	}
	isolateConfig(t)
	t.Setenv("MODEL_CAPABILITIES", path)
	t.Setenv("PROVIDER_URLS", "local="+srv.URL+"/")
	t.Setenv("LOCAL_API_KEY", "test")

	conf, _, err := openai.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, os.Remove(path))
	p, err := openai.NewProvider(conf)
	if !assert.NoError(t, err) {
		return
	}
	models := listModels(t, conf, p)
	assert.Equal(t, []api.AgentModel{{ID: "org/model", Efforts: []api.Effort{"high"}}}, models)
	cfg := &api.AgentConfig{
		Provider: "local",
		Model:    "org/model",
		Effort:   "high",
	}
	c, err := p.Open(&worker.Chat{Config: cfg})
	if !assert.NoError(t, err) {
		return
	}
	turn, err := c.Turn(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, new(float64(0)), turn.Cost)
}

// isolateConfig clears the worker settings a developer's environment might set
func isolateConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"MCP_SERVERS", "ARGYLL_MCP_URL", "MODEL_PRICES", "MODEL_CAPABILITIES",
		"PROVIDER_URLS", "OPENAI_API_KEY",
	} {
		t.Setenv(key, "")
	}
}

// listModels reads the models a worker lists for its only provider
func listModels(t *testing.T, cfg *worker.Config, p worker.Provider) []api.AgentModel {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, builtins.AgentCapabilitiesPath, nil)
	w := httptest.NewRecorder()
	worker.New(cfg, p).Handler().ServeHTTP(w, req)
	var res api.AgentCapabilities
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	if !assert.Len(t, res.Providers, 1) {
		return nil
	}
	return res.Providers[0].Models
}
