package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/lru"
)

// Worker serves the Agent Worker Contract
type Worker struct {
	cfg      *Config
	client   *http.Client
	schemas  *lru.Cache[stepSchemas]
	provider Provider
}

const (
	// DefaultReportTimeout bounds each request reporting an outcome
	DefaultReportTimeout = 30 * time.Second

	// DefaultListTimeout bounds discovery of models and MCP tools
	DefaultListTimeout = 10 * time.Second

	initialReportBackoff = time.Second
	maxReportRetries     = 3
	schemaCacheSize      = 1024
)

var (
	ErrInvalidRequest    = errors.New("invalid agent request")
	ErrMissingWebhook    = errors.New(api.HeaderWebhookURL + " header required")
	ErrMissingAgentSetup = errors.New("provider, model, and prologue are required")
	ErrInvalidLimits     = errors.New("max_turns, max_tokens, and timeout must be positive")
	ErrMissingServerName = errors.New("MCP server name required")
	ErrUnknownProvider   = errors.New("unknown provider")
	ErrReportFailed      = errors.New("outcome report failed")
)

// New returns a worker offering what cfg configures
func New(cfg *Config, provider Provider) *Worker {
	return &Worker{
		cfg:      cfg,
		client:   &http.Client{Timeout: DefaultReportTimeout},
		schemas:  lru.NewCache[stepSchemas](schemaCacheSize),
		provider: provider,
	}
}

// Handler serves the worker's endpoints
func (w *Worker) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+builtins.AgentInvokePath, w.invoke)
	mux.HandleFunc("GET "+builtins.AgentHealthPath, func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET "+builtins.AgentCapabilitiesPath, w.capabilities)
	return mux
}

func (w *Worker) invoke(rw http.ResponseWriter, r *http.Request) {
	webhook := r.Header.Get(api.HeaderWebhookURL)
	if webhook == "" {
		writeProblem(rw, http.StatusBadRequest, ErrMissingWebhook)
		return
	}
	var req api.AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		err = fmt.Errorf("%w: %w", ErrInvalidRequest, err)
		writeProblem(rw, http.StatusBadRequest, err)
		return
	}
	if err := validate(&req); err != nil {
		writeProblem(rw, http.StatusBadRequest, err)
		return
	}
	if err := w.offers(&req); err != nil {
		writeProblem(rw, http.StatusUnprocessableEntity, err)
		return
	}
	rw.WriteHeader(http.StatusAccepted)
	go func() {
		w.report(webhook, w.attempt(context.Background(), &req))
	}()
}

// offers reports whether the worker reaches a request's provider and MCP servers
func (w *Worker) offers(req *api.AgentRequest) error {
	cfg := req.Step.Agent
	if !w.provider.Supports(cfg.Provider) {
		return fmt.Errorf("%w: %s", ErrUnknownProvider, cfg.Provider)
	}
	for _, sel := range cfg.MCP {
		if _, ok := w.cfg.Servers[sel.Server]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownServer, sel.Server)
		}
	}
	return nil
}

// report posts an outcome to the webhook URL, retrying transport errors and 5xx responses
func (w *Worker) report(webhook string, out api.AgentOutcome) {
	for i := 0; ; i++ {
		err := w.post(webhook, out)
		if err == nil {
			return
		}
		slog.Warn("Outcome report failed", log.Error(err))
		if i == maxReportRetries {
			slog.Error("Gave up reporting outcome", slog.String("webhook", webhook))
			return
		}
		time.Sleep(initialReportBackoff << i)
	}
}

func (w *Worker) post(webhook string, out api.AgentOutcome) error {
	req, err := outcomeRequest(webhook, out)
	if err != nil {
		return err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: %s", ErrReportFailed, resp.Status)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		slog.Error("Outcome rejected", slog.Int("status", resp.StatusCode))
	}
	return nil
}

func outcomeRequest(webhook string, out api.AgentOutcome) (*http.Request, error) {
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", api.JSONContentType)
	return req, nil
}

func validate(req *api.AgentRequest) error {
	if req.Step == nil || req.Step.Type != api.StepTypeAgent || req.Step.Agent == nil {
		return ErrInvalidRequest
	}
	cfg := req.Step.Agent
	switch {
	case cfg.Provider == "", cfg.Model == "", cfg.Prologue == "":
		return fmt.Errorf("%w: %w", ErrInvalidRequest, ErrMissingAgentSetup)
	case cfg.MaxTurns < 1, cfg.MaxTokens < 1, req.Timeout < 1:
		return fmt.Errorf("%w: %w", ErrInvalidRequest, ErrInvalidLimits)
	}
	for _, attr := range req.Step.Attributes {
		if attr == nil {
			return ErrInvalidRequest
		}
	}
	for _, sel := range cfg.MCP {
		if sel.Server == "" {
			return fmt.Errorf("%w: %w", ErrInvalidRequest, ErrMissingServerName)
		}
	}
	return nil
}

func writeProblem(rw http.ResponseWriter, status int, err error) {
	rw.Header().Set("Content-Type", api.ProblemJSONContentType)
	rw.WriteHeader(status)
	_ = json.NewEncoder(rw).Encode(api.ProblemDetails{
		Status: status,
		Detail: err.Error(),
	})
}
