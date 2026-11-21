package gemini

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions/models/apierrors"
	"google.golang.org/genai/interactions/models/interactions"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

// Provider runs agent work through the Gemini Interactions API
type Provider struct {
	url    string
	models map[worker.ModelKey]worker.Capabilities
}

// Name identifies the Gemini provider
const Name api.ProviderID = "gemini"

const (
	apiKeyEnv        = "GEMINI_API_KEY"
	generateAction   = "generateContent"
	modelResourceDir = "models/"
)

var (
	ErrUnsupportedModel = errors.New("model does not support agent tool use")
	ErrInvalidEffort    = errors.New("unsupported model effort")
)

var _ worker.Provider = (*Provider)(nil)

// NewProvider returns a Gemini provider using the configured base URL and model capabilities
func NewProvider(cfg *worker.Config) *Provider {
	return &Provider{
		url:    cfg.Providers[Name],
		models: cfg.Capabilities,
	}
}

// Supports reports whether this adapter serves the named provider
func (*Provider) Supports(name api.ProviderID) bool {
	return name == Name
}

// Names lists Gemini when an API key or base URL is configured
func (p *Provider) Names() []api.ProviderID {
	if p.url != "" || os.Getenv(apiKeyEnv) != "" {
		return []api.ProviderID{Name}
	}
	return nil
}

// Models lists the Gemini models that generate content
func (p *Provider) Models(ctx context.Context, _ api.ProviderID) ([]api.AgentModel, error) {
	client, err := p.client(ctx)
	if err != nil {
		return nil, err
	}
	var models []api.AgentModel
	for m, err := range client.Models.All(ctx) {
		if err != nil {
			return models, problem(err)
		}
		if slices.Contains(m.SupportedActions, generateAction) {
			models = append(models, api.AgentModel{
				ID: api.ModelID(strings.TrimPrefix(m.Name, modelResourceDir)),
			})
		}
	}
	return models, nil
}

// Open starts a Gemini conversation with the task's context and tools
func (p *Provider) Open(ch *worker.Chat) (worker.Convo, error) {
	if os.Getenv(apiKeyEnv) == "" {
		return nil, fmt.Errorf("%w: %s", worker.ErrMissingAPIKey, apiKeyEnv)
	}
	cfg := ch.Config
	caps := p.models[worker.ModelKey{
		Provider: Name,
		Model:    cfg.Model,
	}]
	if caps.Hidden {
		return nil, problem(fmt.Errorf("%w: %s", ErrUnsupportedModel, cfg.Model))
	}
	if cfg.Effort != "" && !slices.Contains(caps.Efforts, cfg.Effort) {
		return nil, problem(fmt.Errorf("%w: %s: %s", ErrInvalidEffort, cfg.Model, cfg.Effort))
	}
	client, err := p.client(context.Background())
	if err != nil {
		return nil, err
	}
	return newConvo(client, ch), nil
}

func (p *Provider) client(ctx context.Context) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      os.Getenv(apiKeyEnv),
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: p.url},
	})
}

func problem(err error) error {
	if errors.Is(err, ErrUnsupportedModel) || errors.Is(err, ErrInvalidEffort) {
		return withStatus(err, http.StatusUnprocessableEntity)
	}
	if status := statusOf(err); status != 0 {
		return withStatus(err, status)
	}
	return err
}

// statusOf finds the HTTP status behind a Gemini API error
func statusOf(err error) int {
	if res, ok := errors.AsType[genai.APIError](err); ok {
		return res.Code
	}
	if res, ok := errors.AsType[*apierrors.CreateInteractionClientError](err); ok {
		return responseStatus(res.HTTPMeta.Response)
	}
	if res, ok := errors.AsType[*apierrors.CreateInteractionServerError](err); ok {
		return responseStatus(res.HTTPMeta.Response)
	}
	if res, ok := errors.AsType[*apierrors.ResponseValidationError](err); ok {
		return res.StatusCode
	}
	if res, ok := errors.AsType[*apierrors.APIError](err); ok {
		return res.StatusCode
	}
	return 0
}

func responseStatus(res *http.Response) int {
	if res == nil {
		return 0
	}
	return res.StatusCode
}

func withStatus(err error, status int) error {
	return errors.Join(err, &api.ProblemDetails{
		Status: status,
		Detail: err.Error(),
	})
}

func errorText(errs []interactions.Error) string {
	var res strings.Builder
	for _, e := range errs {
		for _, s := range []*string{e.Code, e.Message} {
			if s != nil && *s != "" {
				res.WriteString(": ")
				res.WriteString(*s)
			}
		}
	}
	return res.String()
}
