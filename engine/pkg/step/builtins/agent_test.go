package builtins_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

const agentTimeout = 5 * time.Second

func TestAgentWorkersURL(t *testing.T) {
	workers := builtins.AgentWorkers{
		"anthropic":          "http://anthropic",
		builtins.AnyProvider: "http://any",
	}
	u, ok := workers.URL("anthropic")
	assert.True(t, ok)
	assert.Equal(t, "http://anthropic", u)

	u, ok = workers.URL("openai")
	assert.True(t, ok)
	assert.Equal(t, "http://any", u)

	_, ok = builtins.AgentWorkers{}.URL("openai")
	assert.False(t, ok)
}

func TestAgentValidate(t *testing.T) {
	callback := builtins.BaseCallbackURL("http://engine")
	workers := builtins.AgentWorkers{"anthropic": "http://worker"}

	h := builtins.Agent(workers, callback, agentTimeout)
	assert.NoError(t, h.Validate(newAgentStep()))
	assert.ErrorIs(t, h.Validate(&api.Step{ID: "s"}), api.ErrAgentRequired)

	assert.ErrorIs(t, h.Validate(openAIAgentStep()), builtins.ErrNoAgentWorker)

	h = builtins.Agent(workers, nil, agentTimeout)
	assert.ErrorIs(t, h.Validate(newAgentStep()), builtins.ErrNoCallbackURL)
}

func TestAgentInvoke(t *testing.T) {
	var got api.AgentRequest
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, builtins.AgentInvokePath, r.URL.Path)
		header = r.Header
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	h := builtins.Agent(
		builtins.AgentWorkers{builtins.AnyProvider: srv.URL},
		builtins.BaseCallbackURL("http://engine"), agentTimeout,
	)
	rt, calls := newRuntime("flow-1", "step-1", api.Metadata{})
	st := newAgentStep()
	err := h.Invoke(rt, st, api.Args{"order": "o-1"}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, calls.completeCalls)

	want := api.AgentRequest{
		Step:    st,
		Inputs:  api.Args{"order": "o-1"},
		Timeout: agentTimeout.Milliseconds(),
	}
	assert.Equal(t, want, got)
	assert.Equal(t, "token-1", header.Get(api.HeaderWorkToken))
	assert.Equal(t, "flow-1", header.Get(api.HeaderFlowID))
	assert.Equal(t,
		"http://engine/callbacks/flow-1/step-1/token-1/agent/"+callbackKey,
		header.Get(api.HeaderWebhookURL))
}

func TestAgentInvokeStepTimeout(t *testing.T) {
	var got api.AgentRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	h := builtins.Agent(
		builtins.AgentWorkers{builtins.AnyProvider: srv.URL},
		builtins.BaseCallbackURL("http://engine"), agentTimeout,
	)
	st := agentStep(func(c api.AgentConfig) api.AgentConfig {
		c.Timeout = 60_000
		return c
	})
	rt, _ := newRuntime("flow-1", "step-1", api.Metadata{})
	assert.NoError(t, h.Invoke(rt, st, api.Args{}, "token-1"))
	assert.Equal(t, int64(60_000), got.Timeout)
}

func TestAgentInvokeSendsEngineCredentials(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "engine"}),
	}}
	h := builtins.AgentWithClient(
		builtins.AgentWorkers{builtins.AnyProvider: srv.URL},
		builtins.BaseCallbackURL("http://engine"), client, agentTimeout,
	)
	rt, _ := newRuntime("flow-1", "step-1", api.Metadata{})
	assert.NoError(t, h.Invoke(rt, newAgentStep(), api.Args{}, "token-1"))
	assert.Equal(t, "Bearer engine", auth)
}

func TestAgentInvokeErrors(t *testing.T) {
	status := http.StatusServiceUnavailable
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()

	callback := builtins.BaseCallbackURL("http://engine")
	h := builtins.Agent(builtins.AgentWorkers{"anthropic": srv.URL}, callback, agentTimeout)
	rt, _ := newRuntime("flow-1", "step-1", api.Metadata{})

	err := h.Invoke(rt, newAgentStep(), api.Args{}, "token-1")
	assert.ErrorIs(t, err, api.ErrWorkNotCompleted)

	status = http.StatusUnprocessableEntity
	err = h.Invoke(rt, newAgentStep(), api.Args{}, "token-1")
	assert.ErrorIs(t, err, builtins.ErrHTTPError)
	assert.NotErrorIs(t, err, api.ErrWorkNotCompleted)

	err = h.Invoke(rt, openAIAgentStep(), api.Args{}, "token-1")
	assert.ErrorIs(t, err, builtins.ErrNoAgentWorker)

	srv.Close()
	err = h.Invoke(rt, newAgentStep(), api.Args{}, "token-1")
	assert.ErrorIs(t, err, api.ErrWorkNotCompleted)
}

func newAgentStep() *api.Step {
	return agentStep(func(c api.AgentConfig) api.AgentConfig {
		return c
	})
}

func openAIAgentStep() *api.Step {
	return agentStep(func(c api.AgentConfig) api.AgentConfig {
		c.Provider = "openai"
		return c
	})
}

// agentStep builds the standard agent step with its config as the given function rewrites it
func agentStep(fn func(api.AgentConfig) api.AgentConfig) *api.Step {
	cfg := fn(api.AgentConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-5-5",
		Prologue: "# Score\n\nScore the `order`",
		MCP: []api.MCPServer{
			{
				Server: "crm",
				Tools:  []string{"lookup"},
			},
		},
		MaxTurns:  4,
		MaxTokens: 1024,
	})
	return &api.Step{
		ID:    callbackStep,
		Name:  "Score",
		Type:  api.StepTypeAgent,
		Agent: &cfg,
		Attributes: api.AttributeSpecs{
			"order": {Role: api.RoleRequired},
			"score": {Role: api.RoleOutput},
			"why": {
				Role: api.RoleOutput,
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{Name: "reason"},
				},
			},
		},
	}
}
