package main_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-bedrock/internal/bedrock"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type seenRequest struct {
	path string
	auth string
}

func TestDefaultProvider(t *testing.T) {
	credentials(t, true)
	_, p, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{bedrock.Name}, p.Names())
	assert.True(t, p.Supports(bedrock.Name))
	assert.False(t, p.Supports("anthropic"))
}

func TestConverse(t *testing.T) {
	credentials(t, true)
	seen := make(chan seenRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- seenRequest{
			path: r.URL.Path,
			auth: r.Header.Get("Authorization"),
		}
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = io.WriteString(w, `{"output":{"message":{"role":"assistant",
			"content":[{"text":"ok"}]}},"stopReason":"end_turn",
			"usage":{"inputTokens":1,"outputTokens":1,"totalTokens":2}}`)
	}))
	defer srv.Close()

	turn, err := converse(t, srv.URL)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), turn.InputTokens)

	req := <-seen
	assert.Equal(t, "/model/m/converse", req.path)
	assert.True(t, strings.HasPrefix(req.auth, "AWS4-HMAC-SHA256 "))
	assert.Contains(t, req.auth, "/us-east-1/bedrock/aws4_request")
}

func TestNoMessage(t *testing.T) {
	credentials(t, true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = io.WriteString(w, `{"output":{},"stopReason":"end_turn",
			"usage":{"inputTokens":3,"outputTokens":2,"totalTokens":5}}`)
	}))
	defer srv.Close()

	turn, err := converse(t, srv.URL)
	assert.True(t, errors.Is(err, bedrock.ErrNoMessage))
	assert.Equal(t, int64(3), turn.InputTokens)
	assert.Equal(t, int64(2), turn.OutputTokens)
}

func converse(t *testing.T, url string) (worker.Turn, error) {
	t.Helper()
	testModels(t)
	t.Setenv("AWS_ENDPOINT_URL", url)
	_, p, err := bedrock.LoadConfig()
	if !assert.NoError(t, err) {
		return worker.Turn{}, err
	}
	c, err := p.Open(&worker.Chat{
		Config: &api.AgentConfig{
			Provider:  bedrock.Name,
			Model:     "m",
			MaxTokens: 16,
		},
		System: "system",
		User:   "user",
		Tools: []worker.ToolSpec{
			{Name: "finish", Schema: map[string]any{"type": "object"}},
		},
	})
	if !assert.NoError(t, err) {
		return worker.Turn{}, err
	}
	return c.Turn(t.Context())
}
