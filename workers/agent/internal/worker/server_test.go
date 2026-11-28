package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	blockingConvo blockingProvider

	blockingProvider struct {
		finish <-chan struct{}
	}
)

func TestServe(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
		port string
	}{
		{name: "configured port", host: "localhost", port: "invalid-port"},
		{name: "default port", host: "::invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOST", tc.host)
			t.Setenv("PORT", tc.port)
			err := worker.Serve(&worker.Config{}, nil)
			assert.Error(t, err)
		})
	}
}

func TestShutdownDrainsAttempt(t *testing.T) {
	finish := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-finish:
		default:
			close(finish)
		}
	})
	reported := make(chan struct{}, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reported <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(callback.Close)

	w := worker.New(&worker.Config{}, blockingProvider{finish: finish})
	h := w.Handler()

	if !assert.Equal(t, http.StatusAccepted, invokeWorker(t, h, callback.URL)) {
		return
	}
	stopped, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, w.Shutdown(stopped), context.Canceled)
	assert.Equal(t, http.StatusServiceUnavailable, invokeWorker(t, h, callback.URL))

	close(finish)
	shutdown, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	assert.NoError(t, w.Shutdown(shutdown))
	assert.Len(t, reported, 1)
}

func TestShutdownCancelsAttempt(t *testing.T) {
	reported := make(chan api.AgentOutcome, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var out api.AgentOutcome
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&out))
		reported <- out
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(callback.Close)

	w := worker.New(&worker.Config{}, blockingProvider{})
	if !assert.Equal(t, http.StatusAccepted, invokeWorker(t, w.Handler(), callback.URL)) {
		return
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	assert.NoError(t, w.Shutdown(ctx))
	if !assert.Len(t, reported, 1) {
		return
	}
	out := <-reported
	if assert.NotNil(t, out.Problem) {
		assert.Equal(t, http.StatusBadGateway, out.Problem.Status)
	}
}

func (blockingProvider) Supports(api.ProviderID) bool {
	return true
}

func (blockingProvider) Names() []api.ProviderID {
	return nil
}

func (blockingProvider) Models(context.Context, api.ProviderID) ([]api.AgentModel, error) {
	return nil, nil
}

func (p blockingProvider) Open(*worker.Chat) (worker.Convo, error) {
	return blockingConvo(p), nil
}

func (c blockingConvo) Turn(ctx context.Context) (worker.Turn, error) {
	select {
	case <-c.finish:
		return worker.Turn{}, errors.New("test model stopped")
	case <-ctx.Done():
		return worker.Turn{}, ctx.Err()
	}
}

func (blockingConvo) Answer([]worker.ToolResult) {}

func (blockingConvo) Nudge(string) {}

func invokeWorker(t *testing.T, h http.Handler, webhook string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, builtins.AgentInvokePath, strings.NewReader(
		`{"step":{"type":"agent","agent":{"provider":"test","model":"test","prologue":"test",`+
			`"max_turns":1,"max_tokens":1}},"timeout":5000}`,
	))
	req.Header.Set(api.HeaderWebhookURL, webhook)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}
