package check_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll-agent-check/internal/check"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

// fakeWorker serves the worker contract, reporting each attempt with outcome
type fakeWorker struct {
	outcome func(webhook string)
}

func TestRun(t *testing.T) {
	t.Run("conforming worker passes", func(t *testing.T) {
		body := `{
			"outputs":{"echo":"argyll"},
			"usage":{"input_tokens":10,"output_tokens":2,"cost":0.001}
		}`
		res := run(t, &fakeWorker{outcome: report(body)})
		var failed []check.Result
		for _, r := range res {
			if r.Err != nil {
				failed = append(failed, r)
			}
		}
		assert.Empty(t, failed)
		assert.Len(t, res, 4)
	})

	t.Run("non-envelope callback fails", func(t *testing.T) {
		res := run(t, &fakeWorker{outcome: report(`{"echo":"argyll"}`)})
		assert.ErrorIs(t, res[2].Err, check.ErrInvalidOutcome)
		assert.ErrorIs(t, res[3].Err, check.ErrInvalidOutcome)
	})

	t.Run("malformed callback fails", func(t *testing.T) {
		res := run(t, &fakeWorker{outcome: report(`{`)})
		assert.ErrorIs(t, res[2].Err, check.ErrInvalidOutcome)
	})

	t.Run("rejected request fails", func(t *testing.T) {
		res := run(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		assert.ErrorIs(t, res[0].Err, check.ErrUnhealthy)
		assert.ErrorIs(t, res[1].Err, check.ErrNotAccepted)
	})

	t.Run("invalid callback address fails", func(t *testing.T) {
		_, err := check.Run(&check.Config{Listen: "localhost:invalid"})
		assert.Error(t, err)
	})

	t.Run("wrong outputs and missing usage fail", func(t *testing.T) {
		res := run(t, &fakeWorker{outcome: report(`{"outputs":{"echo":"argyll","extra":1}}`)})
		assert.ErrorIs(t, res[2].Err, check.ErrOutputMismatch)
		assert.ErrorIs(t, res[3].Err, check.ErrMissingUsage)
	})

	t.Run("problem outcome fails with its usage checked", func(t *testing.T) {
		body := `{"problem":{"status":401,"detail":"bad key"},"usage":{"input_tokens":0}}`
		res := run(t, &fakeWorker{outcome: report(body)})
		assert.ErrorIs(t, res[2].Err, check.ErrProblemOutcome)
		assert.ErrorIs(t, res[3].Err, check.ErrMissingUsage)
	})

	t.Run("silent worker fails outcome", func(t *testing.T) {
		if testing.Short() {
			t.Skip("waits out the report grace period")
		}
		res := run(t, &fakeWorker{outcome: func(string) {}})
		assert.ErrorIs(t, res[2].Err, check.ErrNoOutcome)
		assert.ErrorIs(t, res[3].Err, check.ErrNoOutcome)
	})
}

func (f *fakeWorker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == builtins.AgentHealthPath {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var req api.AgentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	webhook := r.Header.Get(api.HeaderWebhookURL)
	invalid := err != nil || req.Step == nil || req.Step.Agent == nil || webhook == ""
	if !invalid {
		cfg := req.Step.Agent
		invalid = cfg.Provider == "" || len(cfg.MCP) > 0
	}
	if invalid {
		w.Header().Set("Content-Type", api.ProblemJSONContentType)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	go f.outcome(webhook)
}

func run(t *testing.T, handler http.Handler) []check.Result {
	t.Helper()
	srv := httptest.NewServer(handler)
	defer srv.Close()
	res, err := check.Run(&check.Config{
		WorkerURL: srv.URL,
		Provider:  "anthropic",
		Model:     "claude-sonnet-5-5",
		Listen:    "localhost:0",
		Timeout:   10 * time.Millisecond,
	})
	assert.NoError(t, err)
	return res
}

func report(body string) func(string) {
	return func(webhook string) {
		req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBufferString(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", api.JSONContentType)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
}
