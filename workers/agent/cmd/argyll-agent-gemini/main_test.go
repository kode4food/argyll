package main_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-gemini/internal/gemini"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type seenRequest struct {
	path string
	key  string
	body map[string]any
}

// toolTurn is a reply with a thought, whose signature must come back, and a function call
const toolTurn = `{"id":"i1","status":"requires_action","steps":[
	{"type":"thought","signature":"sig-1"},
	{"type":"function_call","id":"c1","name":"lookup","arguments":{"name":"acme"}}
],"usage":{"total_input_tokens":7,"total_cached_tokens":2,"total_output_tokens":3,
	"total_thought_tokens":4,"total_tool_use_tokens":1}}`

func TestDefaultProvider(t *testing.T) {
	credentials(t, true)
	_, p, err := gemini.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []api.ProviderID{gemini.Name}, p.Names())
	assert.True(t, p.Supports(gemini.Name))
	assert.False(t, p.Supports("openai"))

	credentials(t, false)
	assert.Empty(t, p.Names())
}

func TestInteraction(t *testing.T) {
	credentials(t, true)
	testModels(t)
	var mu sync.Mutex
	var seen []seenRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		seen = append(seen, seenRequest{
			path: r.URL.Path,
			key:  r.Header.Get("x-goog-api-key"),
			body: body,
		})
		mu.Unlock()
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = io.WriteString(w, toolTurn)
	}))
	defer srv.Close()

	c := open(t, srv.URL, "high")
	if c == nil {
		return
	}
	turn, err := c.Turn(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, int64(8), turn.InputTokens)
	assert.Equal(t, int64(7), turn.OutputTokens)
	assert.Equal(t, int64(2), turn.CacheInputTokens)
	if assert.Len(t, turn.Calls, 1) {
		assert.Equal(t, "c1", turn.Calls[0].ID)
		assert.Equal(t, "lookup", turn.Calls[0].Name)
		assert.JSONEq(t, `{"name":"acme"}`, string(turn.Calls[0].Args))
	}
	c.Answer([]worker.ToolResult{{
		CallID:  "c1",
		Content: "acme has 40 seats",
	}})
	_, err = c.Turn(t.Context())
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	if !assert.Len(t, seen, 2) {
		return
	}
	first := seen[0]
	assert.Equal(t, "/v1/interactions", first.path)
	assert.Equal(t, "test", first.key)
	assert.Equal(t, false, first.body["store"])
	assert.Equal(t, "gemini-first", first.body["model"])
	assert.Equal(t, "system", first.body["system_instruction"])
	assert.Equal(t, map[string]any{
		"max_output_tokens": float64(16),
		"thinking_level":    "high",
	}, first.body["generation_config"])

	input, _ := seen[1].body["input"].([]any)
	if assert.Len(t, input, 4) {
		assert.Equal(t, "user_input", step(input[0])["type"])
		assert.Equal(t, "sig-1", step(input[1])["signature"])
		assert.Equal(t, "function_call", step(input[2])["type"])
		result := step(input[3])
		assert.Equal(t, "function_result", result["type"])
		assert.Equal(t, "c1", result["call_id"])
		assert.Equal(t, "lookup", result["name"])
		assert.Equal(t, "acme has 40 seats", result["result"])
	}
}

func TestRejects(t *testing.T) {
	credentials(t, true)
	testModels(t)
	_, p, err := gemini.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	for _, tc := range []struct {
		model  api.ModelID
		effort api.Effort
		err    error
	}{
		{model: "gemini-tts", err: gemini.ErrUnsupportedModel},
		{model: "gemini-first", effort: "xhigh", err: gemini.ErrInvalidEffort},
		{model: "gemini-second", effort: "low", err: gemini.ErrInvalidEffort},
	} {
		_, err := p.Open(chat(tc.model, tc.effort))
		assert.True(t, errors.Is(err, tc.err))
		if problem, ok := errors.AsType[*api.ProblemDetails](err); assert.True(t, ok) {
			assert.Equal(t, http.StatusUnprocessableEntity, problem.Status)
		}
	}
}

func open(t *testing.T, url string, effort api.Effort) worker.Convo {
	t.Helper()
	t.Setenv("PROVIDER_URLS", "gemini="+url)
	_, p, err := gemini.LoadConfig()
	if !assert.NoError(t, err) {
		return nil
	}
	c, err := p.Open(chat("gemini-first", effort))
	if !assert.NoError(t, err) {
		return nil
	}
	return c
}

func chat(model api.ModelID, effort api.Effort) *worker.Chat {
	return &worker.Chat{
		Config: &api.AgentConfig{
			Provider:  gemini.Name,
			Model:     model,
			Effort:    effort,
			MaxTokens: 16,
		},
		System: "system",
		User:   "user",
		Tools: []worker.ToolSpec{
			{Name: "lookup", Schema: map[string]any{"type": "object"}},
		},
	}
}

func step(v any) map[string]any {
	res, _ := v.(map[string]any)
	return res
}
