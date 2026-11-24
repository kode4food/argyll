package worker_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

// tokenRoles authenticates a bearer token as a caller holding the mapped Role
type tokenRoles map[string]auth.Role

func TestHandlerRoles(t *testing.T) {
	keys := tokenRoles{
		"ad": auth.RoleAdmin,
		"op": auth.RoleOperator,
		"rd": auth.RoleReader,
	}
	h := worker.New(&worker.Config{Auth: keys}, nil).Handler()

	for _, tc := range []struct {
		name   string
		method string
		path   string
		key    string
		status int
	}{
		{
			name:   "health is public",
			method: http.MethodGet,
			path:   builtins.AgentHealthPath,
			status: http.StatusNoContent,
		},
		{
			name:   "invoke needs a token",
			method: http.MethodPost,
			path:   builtins.AgentInvokePath,
			status: http.StatusUnauthorized,
		},
		{
			name:   "chat needs a token",
			method: http.MethodPost,
			path:   contract.Path,
			status: http.StatusUnauthorized,
		},
		{
			name:   "reader cannot invoke",
			method: http.MethodPost,
			path:   builtins.AgentInvokePath,
			key:    "rd",
			status: http.StatusForbidden,
		},
		{
			name:   "reader cannot chat",
			method: http.MethodPost,
			path:   contract.Path,
			key:    "rd",
			status: http.StatusForbidden,
		},
		{
			// Agent work carries its own Step definition
			name:   "operator cannot invoke",
			method: http.MethodPost,
			path:   builtins.AgentInvokePath,
			key:    "op",
			status: http.StatusForbidden,
		},
		{
			// Reaches the handler, which wants a webhook URL
			name:   "admin invokes",
			method: http.MethodPost,
			path:   builtins.AgentInvokePath,
			key:    "ad",
			status: http.StatusBadRequest,
		},
		{
			// Reaches the handler, which wants a chat body
			name:   "operator chats",
			method: http.MethodPost,
			path:   contract.Path,
			key:    "op",
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.key != "" {
				req.Header.Set("Authorization", "Bearer "+tc.key)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			assert.Equal(t, tc.status, w.Code)
		})
	}
}

func (t tokenRoles) Authenticate(r *http.Request) (*auth.Principal, error) {
	role, ok := t[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	return &auth.Principal{Roles: []auth.Role{role}}, nil
}
