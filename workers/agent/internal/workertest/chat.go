package workertest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

func (s *suite) testChat(t *testing.T) {
	t.Run("answers with argyll tools", func(t *testing.T) {
		model := &fakeModel{responses: []string{
			s.reply(ToolCall{
				ID:        "t1",
				Name:      worker.ChatServer + "__lookup",
				Arguments: `{"name": "acme"}`,
			}),
			s.reply(),
			s.reply(ToolCall{
				ID:        "t2",
				Name:      "reply",
				Arguments: `{}`,
			}),
			s.reply(ToolCall{
				ID:        "t3",
				Name:      "reply",
				Arguments: `{"text": "40 seats"}`,
			}),
		}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()
		crm := crmServer(t)
		defer crm.Close()
		cfg := s.config(modelSrv.URL)
		cfg.Servers = s.chatServers(crm.URL)

		resp, body := s.postChat(t, cfg, s.chatRequest())
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var res contract.AgentChatReply
		assert.NoError(t, json.Unmarshal(body, &res))
		assert.Equal(t, "40 seats", res.Reply)
		assert.Equal(t, float64(4*replyInput), res.Usage["input_tokens"])

		first := model.request(0)
		tools := mustJSON(t, s.h.Tools(first))
		assert.Contains(t, tools, `"argyll__lookup"`)
		assert.Contains(t, tools, `"reply"`)
		assert.NotContains(t, tools, `"crm__lookup"`)
		assert.Contains(t, mustJSON(t, first), "how many seats")
		assert.Contains(t, mustJSON(t, first), `\"space_id\":\"sales\"`)
		assert.Contains(t, mustJSON(t, s.h.System(first)), "Be brief")
		assert.Contains(t, mustJSON(t, model.request(1)), "acme has 40 seats")
		assert.Contains(t, mustJSON(t, model.request(2)), "Call the `reply` tool")
		assert.Contains(t, mustJSON(t, model.request(3)), "reply takes a non-empty text")
	})

	t.Run("tools act as the caller", func(t *testing.T) {
		model := &fakeModel{responses: []string{s.reply(ToolCall{
			ID:        "t1",
			Name:      "reply",
			Arguments: `{"text": "hi"}`,
		})}}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()
		crm := crmServer(t)
		defer crm.Close()
		cfg := s.config(modelSrv.URL)
		cfg.Servers = map[string]worker.MCPServer{
			worker.ChatServer: {URL: crm.URL},
		}

		// The server accepts only the caller's token, which the worker passes on
		resp, _ := s.postChatAs(t, cfg, s.chatRequest(), "Bearer "+crmToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, _ = s.postChatAs(t, cfg, s.chatRequest(), "Bearer other")
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)

		// The worker's own credentials never stand in for a chat caller
		cfg.Servers = map[string]worker.MCPServer{
			worker.ChatServer: {
				URL:    crm.URL,
				Tokens: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: crmToken}),
			},
		}
		resp, _ = s.postChatAs(t, cfg, s.chatRequest(), "")
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)

		// Nor do the server's configured headers
		cfg.Servers = s.chatServers(crm.URL)
		resp, _ = s.postChatAs(t, cfg, s.chatRequest(), "")
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("rejects", func(t *testing.T) {
		crm := crmServer(t)
		defer crm.Close()
		cfg := s.config("http://localhost:1")
		cfg.Servers = s.chatServers(crm.URL)
		bare := s.config("http://localhost:1")

		unknown := s.chatRequest()
		unknown.Provider = "unknown"
		empty := s.chatRequest()
		empty.Messages = nil
		for _, tc := range []struct {
			name   string
			cfg    *worker.Config
			body   any
			status int
		}{
			{
				name:   "invalid JSON",
				cfg:    cfg,
				body:   "{",
				status: http.StatusBadRequest,
			},
			{
				name:   "no messages",
				cfg:    cfg,
				body:   empty,
				status: http.StatusBadRequest,
			},
			{
				name:   "unknown provider",
				cfg:    cfg,
				body:   unknown,
				status: http.StatusUnprocessableEntity,
			},
			{
				name:   "no argyll server",
				cfg:    bare,
				body:   s.chatRequest(),
				status: http.StatusUnprocessableEntity,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resp, _ := s.postChat(t, tc.cfg, tc.body)
				assert.Equal(t, tc.status, resp.StatusCode)
				assert.Equal(t, api.ProblemJSONContentType, resp.Header.Get("Content-Type"))
			})
		}
	})

	t.Run("provider failure is a problem", func(t *testing.T) {
		model := &fakeModel{status: http.StatusUnauthorized}
		modelSrv := httptest.NewServer(model)
		defer modelSrv.Close()
		crm := crmServer(t)
		defer crm.Close()
		cfg := s.config(modelSrv.URL)
		cfg.Servers = s.chatServers(crm.URL)

		resp, body := s.postChat(t, cfg, s.chatRequest())
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, api.ProblemJSONContentType, resp.Header.Get("Content-Type"))
		assert.Contains(t, string(body), `"status":401`)
	})
}

func (s *suite) chatServers(url string) map[string]worker.MCPServer {
	return map[string]worker.MCPServer{
		worker.ChatServer: crmConfig(url)["crm"],
	}
}

func (s *suite) chatRequest() *contract.AgentChatRequest {
	return &contract.AgentChatRequest{
		Provider: s.h.Provider,
		Model:    testModel,
		Messages: []contract.ChatMessage{
			{Role: contract.ChatRoleUser, Content: "hello"},
			{Role: contract.ChatRoleAssistant, Content: "hi"},
			{Role: contract.ChatRoleUser, Content: "how many seats for acme?"},
		},
		View: &contract.ChatView{SpaceID: "sales", Locale: "de-CH"},
	}
}

// postChat posts a chat as a caller the test MCP server accepts
func (s *suite) postChat(t *testing.T, cfg *worker.Config, body any) (*http.Response, []byte) {
	t.Helper()
	return s.postChatAs(t, cfg, body, "Bearer "+crmToken)
}

// postChatAs posts a chat with the caller's Authorization header, when one is given
func (s *suite) postChatAs(
	t *testing.T, cfg *worker.Config, body any, auth string,
) (*http.Response, []byte) {
	t.Helper()
	srv := httptest.NewServer(s.newWorker(cfg).Handler())
	defer srv.Close()
	data, ok := body.(string)
	if !ok {
		data = mustJSON(t, body)
	}
	req, err := http.NewRequest(
		http.MethodPost, srv.URL+contract.Path, bytes.NewBufferString(data),
	)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", api.JSONContentType)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}
