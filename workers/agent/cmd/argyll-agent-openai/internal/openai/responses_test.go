package openai_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/cmd/argyll-agent-openai/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func TestResponsesEffort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/responses", r.URL.Path)
		var req struct {
			Input     []map[string]any
			Reasoning struct{ Effort string }
			Store     *bool
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) ||
			!assert.NotNil(t, req.Store) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "low", req.Reasoning.Effort)
		assert.False(t, *req.Store)
		w.Header().Set("Content-Type", api.JSONContentType)
		if len(req.Input) == 2 {
			_, _ = io.WriteString(w, `{"output":[`+
				`{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"opaque"},`+
				`{"type":"function_call","id":"fc_1","call_id":"call_1",`+
				`"name":"lookup","arguments":"{}"}]}`)
			return
		}
		if assert.Len(t, req.Input, 5) {
			assert.Equal(t, "opaque", req.Input[2]["encrypted_content"])
			assert.Equal(t, "call_1", req.Input[3]["call_id"])
			assert.Equal(t, "function_call_output", req.Input[4]["type"])
			assert.Equal(t, "call_1", req.Input[4]["call_id"])
		}
		_, _ = io.WriteString(w, `{"output":[{"type":"function_call","call_id":"call_2",`+
			`"name":"finish","arguments":"{}"}],"usage":{"input_tokens":1000,"output_tokens":9,`+
			`"input_tokens_details":{"cached_tokens":300,"cache_write_tokens":200}}}`)
	}))
	defer srv.Close()
	t.Setenv("OPENAI_BASE_URL", srv.URL+"/")
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("MODEL_CAPABILITIES", "")
	p, err := openai.New(nil)
	if !assert.NoError(t, err) {
		return
	}
	cfg := &api.AgentConfig{
		Provider: openai.Name,
		Model:    "gpt-5.6-luna",
		Effort:   "low",
	}
	ch := &worker.Chat{
		Config: cfg,
		System: "System",
		User:   "Input",
	}
	c, err := p.Open(ch)
	if !assert.NoError(t, err) {
		return
	}
	first, err := c.Turn(t.Context())
	if !assert.NoError(t, err) || !assert.Len(t, first.Calls, 1) {
		return
	}
	assert.Equal(t, "lookup", first.Calls[0].Name)

	c.Answer([]worker.ToolResult{{CallID: first.Calls[0].ID, Content: "result"}})
	second, err := c.Turn(t.Context())
	if !assert.NoError(t, err) || !assert.Len(t, second.Calls, 1) {
		return
	}
	assert.Equal(t, "finish", second.Calls[0].Name)
	assert.Equal(t, int64(300), second.CacheInputTokens)
	assert.Equal(t, int64(200), second.CacheOutputTokens)
}

func TestFailedResponse(t *testing.T) {
	for _, tc := range []struct {
		code   string
		status int
	}{
		{code: "server_error", status: http.StatusBadGateway},
		{code: "rate_limit_exceeded", status: http.StatusTooManyRequests},
		{code: "invalid_prompt", status: http.StatusBadRequest},
	} {
		t.Run(tc.code, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/responses", r.URL.Path)
				w.Header().Set("Content-Type", api.JSONContentType)
				_, _ = io.WriteString(w, `{"status":"failed","error":{"code":"`+
					tc.code+`","message":"generation failed"},"output":[],`+
					`"usage":{"input_tokens":23,"output_tokens":2}}`)
			})
			srv := httptest.NewServer(h)
			defer srv.Close()
			t.Setenv("OPENAI_BASE_URL", srv.URL+"/")
			t.Setenv("OPENAI_API_KEY", "test")
			t.Setenv("MODEL_CAPABILITIES", "")
			p, err := openai.New(nil)
			if !assert.NoError(t, err) {
				return
			}
			cfg := &api.AgentConfig{
				Provider: openai.Name,
				Model:    "gpt-5.6-luna",
			}
			c, err := p.Open(&worker.Chat{Config: cfg})
			if !assert.NoError(t, err) {
				return
			}
			turn, err := c.Turn(t.Context())
			want := &api.ProblemDetails{
				Status: tc.status,
				Title:  openai.ErrResponseFailed.Error(),
				Detail: "generation failed",
			}
			assert.Equal(t, want, err)
			assert.Empty(t, turn.Calls)
			assert.Equal(t, int64(23), turn.InputTokens)
			assert.Equal(t, int64(2), turn.OutputTokens)
		})
	}
}
