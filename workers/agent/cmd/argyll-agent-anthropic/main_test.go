package main_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/anthropic"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type seenRequest struct {
	auth   string
	apiKey string
	body   string
}

const messageReply = `{
	"id": "msg_1",
	"type": "message",
	"role": "assistant",
	"model": "m",
	"content": [{"type": "text", "text": "ok"}],
	"stop_reason": "end_turn",
	"usage": {"input_tokens": 1, "output_tokens": 1}
}`

func TestDefaultProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("PREFIX_THRESHOLDS", "")
	p, err := anthropic.NewProvider(map[api.ProviderID]string{"local": "http://localhost:1"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{anthropic.Name}, p.Names())
	assert.True(t, p.Supports(anthropic.Name))
	assert.False(t, p.Supports("openai"))
	assert.False(t, p.Supports("local"))
}

func TestAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "gateway")
	t.Setenv("PREFIX_THRESHOLDS", "")
	seen := make(chan seenRequest, 1)
	srv := httptest.NewServer(recordRequest(seen))
	defer srv.Close()

	p, err := anthropic.NewProvider(map[api.ProviderID]string{anthropic.Name: srv.URL})
	if !assert.NoError(t, err) {
		return
	}
	_, err = p.Models(context.Background(), anthropic.Name)
	assert.NoError(t, err)

	req := <-seen
	assert.Equal(t, "test", req.apiKey)
	assert.Empty(t, req.auth)
}

func TestPromptCache(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	t.Setenv("PREFIX_THRESHOLDS", "")
	seen := make(chan seenRequest, 1)
	srv := httptest.NewServer(recordRequest(seen))
	defer srv.Close()

	p, err := anthropic.NewProvider(map[api.ProviderID]string{anthropic.Name: srv.URL})
	if !assert.NoError(t, err) {
		return
	}
	turn(t, p)

	req := <-seen
	assert.Contains(t, req.body, `"cache_control":{"type":"ephemeral"}`)
}

func TestAuthToken(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "gateway")
	t.Setenv("PREFIX_THRESHOLDS", "")
	seen := make(chan seenRequest, 1)
	srv := httptest.NewServer(recordRequest(seen))
	defer srv.Close()

	p, err := anthropic.NewProvider(map[api.ProviderID]string{anthropic.Name: srv.URL})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{anthropic.Name}, p.Names())
	turn(t, p)

	req := <-seen
	assert.Equal(t, "Bearer gateway", req.auth)
	assert.Empty(t, req.apiKey)
}

func turn(t *testing.T, p *anthropic.Provider) {
	t.Helper()
	c, err := p.Open(&worker.Chat{
		Config: &api.AgentConfig{
			Provider:  anthropic.Name,
			Model:     "m",
			MaxTokens: 16,
		},
		System: "system",
		User:   "user",
	})
	if !assert.NoError(t, err) {
		return
	}
	_, err = c.Turn(context.Background())
	assert.NoError(t, err)
}

func recordRequest(seen chan<- seenRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- seenRequest{
			auth:   r.Header.Get("Authorization"),
			apiKey: r.Header.Get("X-Api-Key"),
			body:   string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(messageReply))
	}
}
