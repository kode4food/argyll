// Package chat forwards UI conversations to agent workers
package chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

// Server relays each chat to the worker serving its provider
type Server struct {
	workers builtins.AgentWorkers
	client  *http.Client
}

const (
	// Path is where the service answers a chat
	Path = contract.Path

	// HealthPath reports that the service is running
	HealthPath = "/health"

	// DefaultTimeout bounds a chat relayed to an agent worker
	DefaultTimeout = 3 * time.Minute
)

var (
	ErrInvalidChat     = errors.New("invalid chat request")
	ErrAgentChatFailed = errors.New("agent worker chat failed")
)

// NewServer relays chats to the given workers through client
func NewServer(workers builtins.AgentWorkers, client *http.Client) *Server {
	return &Server{
		workers: workers,
		client:  client,
	}
}

// Handler serves chat and health
func (s *Server) Handler(a auth.Authenticator) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST "+Path, auth.Handler(http.HandlerFunc(s.relay), a, auth.RoleOperator))
	mux.HandleFunc("GET "+HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// relay forwards a chat to the worker serving its provider and returns the worker's response as is
func (s *Server) relay(w http.ResponseWriter, r *http.Request) {
	var req contract.AgentChatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err == nil {
		err = req.Validate()
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %w", ErrInvalidChat, err))
		return
	}
	base, ok := s.workers.URL(req.Provider)
	if !ok {
		err := fmt.Errorf("%w: %s", builtins.ErrNoAgentWorker, req.Provider)
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	body, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out, err := http.NewRequestWithContext(
		r.Context(), http.MethodPost, base+contract.Path, bytes.NewReader(body),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out.Header.Set("Content-Type", api.JSONContentType)
	// The chat's tools act as the caller, so the worker passes this on to the Argyll MCP server
	if authz := r.Header.Get("Authorization"); authz != "" {
		out.Header.Set("Authorization", authz)
	}
	resp, err := s.client.Do(out)
	if err != nil {
		slog.Warn("Agent chat relay failed", slog.String("worker", base), log.Error(err))
		writeError(w, http.StatusBadGateway, fmt.Errorf("%w: %w", ErrAgentChatFailed, err))
		return
	}
	defer func() { _ = resp.Body.Close() }()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", api.JSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.ErrorResponse{
		Error:  err.Error(),
		Status: status,
	})
}
