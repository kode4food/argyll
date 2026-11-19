package main_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-openai/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
	"github.com/kode4food/argyll/workers/agent/internal/workertest"
)

func TestWorker(t *testing.T) {
	t.Setenv("MODEL_CAPABILITIES", "")
	workertest.Run(t, openai.Name, func(cfg *worker.Config) worker.Provider {
		p, err := openai.New(cfg.Providers)
		assert.NoError(t, err)
		return p
	})
}

func TestDefaultProvider(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("MODEL_CAPABILITIES", "")
	p, err := openai.New(nil)
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
	p, err := openai.New(map[api.ProviderID]string{"anthropic": "http://gateway"})
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, p.Supports("anthropic"))
	assert.Equal(t, []api.ProviderID{"anthropic"}, p.Names())
}

func TestModelEfforts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"gpt-5.4-nano"},`+
				`{"id":"gpt-5.4-nano-2026-03-17"},{"id":"new-model"}]}`)
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
	t.Setenv("OPENAI_BASE_URL", srv.URL+"/")
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("MODEL_CAPABILITIES", "")

	p, err := openai.New(nil)
	if !assert.NoError(t, err) {
		return
	}
	models, err := p.Models(t.Context(), openai.Name)
	assert.NoError(t, err)
	efforts := []api.Effort{"none", "low", "medium", "high", "xhigh"}
	assert.Equal(t, []api.AgentModel{
		{ID: "gpt-5.4-nano", Efforts: efforts},
		{ID: "gpt-5.4-nano-2026-03-17", Efforts: efforts},
		{ID: "new-model"},
	}, models)
	for _, model := range []api.ModelID{"gpt-5.4-nano", "new-model"} {
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
}

func TestCapabilityOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"org/model"}]}`)
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
	data := []byte(`{"local/org/model":{"endpoint":"responses","efforts":["high"]}}`)
	if !assert.NoError(t, os.WriteFile(path, data, 0o600)) {
		return
	}
	t.Setenv("MODEL_CAPABILITIES", path)
	t.Setenv("LOCAL_API_KEY", "test")

	p, err := openai.New(map[api.ProviderID]string{"local": srv.URL + "/"})
	if !assert.NoError(t, err) {
		return
	}
	models, err := p.Models(t.Context(), "local")
	assert.NoError(t, err)
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
