package builtins

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// AgentWorkers maps each agent provider to the base URL of the worker serving it. AnyProvider
	// names the worker serving every provider not listed
	AgentWorkers map[api.ProviderID]string

	agentHandler struct {
		workers  AgentWorkers
		callback CallbackURL
		client   *httpClient
	}
)

const (
	// AnyProvider is the AgentWorkers key for the worker serving unlisted providers
	AnyProvider = "*"

	// AgentInvokePath is where a worker accepts agent work, below its base URL
	AgentInvokePath = "/invoke"

	// AgentHealthPath is where a worker reports its health, below its base URL
	AgentHealthPath = "/health"

	// AgentCapabilitiesPath is where a worker lists its providers, models, and MCP servers
	AgentCapabilitiesPath = "/capabilities"
)

var (
	ErrNoAgentWorker = errors.New("no worker serves agent provider")
)

// Agent returns the standard agent handler. Each attempt is dispatched to the worker serving the
// step's provider, which reports the outcome through the callback URL
func Agent(workers AgentWorkers, callback CallbackURL, timeout time.Duration) *step.Handler {
	a := &agentHandler{
		workers:  workers,
		callback: callback,
		client: &httpClient{
			httpClient: &http.Client{},
			timeout:    timeout,
		},
	}
	return &step.Handler{
		Validate: a.validate,
		Invoke:   a.invoke,
	}
}

// URL returns the base URL of the worker serving a provider
func (w AgentWorkers) URL(provider api.ProviderID) (string, bool) {
	if u, ok := w[provider]; ok {
		return u, true
	}
	u, ok := w[AnyProvider]
	return u, ok
}

func (a *agentHandler) validate(st *api.Step) error {
	if st.Agent == nil {
		return api.ErrAgentRequired
	}
	if a.callback == nil {
		return fmt.Errorf("%w: %s", ErrNoCallbackURL, st.ID)
	}
	if _, ok := a.workers.URL(st.Agent.Provider); !ok {
		return fmt.Errorf("%w: %s", ErrNoAgentWorker, st.Agent.Provider)
	}
	return nil
}

func (a *agentHandler) invoke(
	rt step.Runtime, st *api.Step, inputs api.Args, token api.Token,
) error {
	base, ok := a.workers.URL(st.Agent.Provider)
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoAgentWorker, st.Agent.Provider)
	}
	meta := metadata(rt, token)
	meta[api.MetaWebhookURL] = a.callback(api.FlowStep{
		FlowID: rt.FlowID(),
		StepID: rt.StepID(),
	}, token, api.ActionAgent)

	req := a.request(st, applyMetaInputs(st, inputs, meta))
	return a.client.postJSON(st, base+AgentInvokePath, req, meta)
}

func (a *agentHandler) request(st *api.Step, inputs api.Args) api.AgentRequest {
	timeout := st.Agent.Timeout
	if timeout == 0 {
		timeout = a.client.timeout.Milliseconds()
	}
	return api.AgentRequest{
		Step:    st,
		Inputs:  inputs,
		Timeout: timeout,
	}
}
