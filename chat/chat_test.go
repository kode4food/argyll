package chat_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/chat"
	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type reader struct{}

const chatAuthorization = "Bearer user"

func TestChat(t *testing.T) {
	req := contract.AgentChatRequest{
		Provider: "anthropic",
		Model:    "claude",
		Messages: []contract.ChatMessage{{Role: contract.ChatRoleUser, Content: "hi"}},
	}
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, contract.Path, r.URL.Path)
		assert.Equal(t, chatAuthorization, r.Header.Get("Authorization"))
		var got contract.AgentChatRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, req, got)
		w.Header().Set("Content-Type", api.JSONContentType)
		_ = json.NewEncoder(w).Encode(contract.AgentChatReply{
			Reply: "hello",
			Usage: api.Usage{"input_tokens": 3},
		})
	}))
	defer worker.Close()
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", api.ProblemJSONContentType)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(api.NewProblem(http.StatusUnauthorized, "no key"))
	}))
	defer failing.Close()

	t.Run("relays the reply", func(t *testing.T) {
		w := postChat(builtins.AgentWorkers{"*": worker.URL}, mustJSON(t, req))
		assert.Equal(t, http.StatusOK, w.Code)
		var res contract.AgentChatReply
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, "hello", res.Reply)
		assert.Equal(t, float64(3), res.Usage["input_tokens"])
	})

	t.Run("relays worker problems", func(t *testing.T) {
		w := postChat(builtins.AgentWorkers{"anthropic": failing.URL}, mustJSON(t, req))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, api.ProblemJSONContentType, w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "no key")
	})

	t.Run("rejects", func(t *testing.T) {
		invalid := req
		invalid.Messages = nil
		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()
		for _, tc := range []struct {
			name    string
			workers builtins.AgentWorkers
			body    string
			status  int
		}{
			{
				name:   "invalid JSON",
				body:   "{",
				status: http.StatusBadRequest,
			},
			{
				name:   "invalid request",
				body:   mustJSON(t, invalid),
				status: http.StatusBadRequest,
			},
			{
				name:   "no worker",
				body:   mustJSON(t, req),
				status: http.StatusUnprocessableEntity,
			},
			{
				name:    "unreachable worker",
				workers: builtins.AgentWorkers{"*": closed.URL},
				body:    mustJSON(t, req),
				status:  http.StatusBadGateway,
			},
			{
				name:    "invalid worker URL",
				workers: builtins.AgentWorkers{"*": "http://bad host"},
				body:    mustJSON(t, req),
				status:  http.StatusInternalServerError,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				w := postChat(tc.workers, tc.body)
				assert.Equal(t, tc.status, w.Code)
				var res api.ErrorResponse
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
				assert.Equal(t, tc.status, res.Status)
			})
		}
	})
}

func TestChatAuthentication(t *testing.T) {
	h := chat.NewServer(nil, http.DefaultClient).Handler(reader{})
	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{
			method: http.MethodPost,
			path:   chat.Path,
			status: http.StatusForbidden,
		},
		{
			method: http.MethodGet,
			path:   chat.HealthPath,
			status: http.StatusNoContent,
		},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		assert.Equal(t, tc.status, w.Code)
	}
}

func (reader) Authenticate(*http.Request) (*auth.Principal, error) {
	return &auth.Principal{Roles: []auth.Role{auth.RoleReader}}, nil
}

func postChat(workers builtins.AgentWorkers, body string) *httptest.ResponseRecorder {
	h := chat.NewServer(workers, http.DefaultClient).Handler(nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, chat.Path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", chatAuthorization)
	h.ServeHTTP(w, req)
	return w
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return string(data)
}
