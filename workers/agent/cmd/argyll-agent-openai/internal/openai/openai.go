package openai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	// Provider runs agent work through OpenAI-compatible APIs
	Provider struct {
		urls   map[api.ProviderID]string
		models map[worker.ModelKey]modelCapabilities
	}

	openAIConvo struct {
		client openai.Client
		params openai.ChatCompletionNewParams
	}
)

// Name identifies the default OpenAI provider
const Name api.ProviderID = "openai"

var (
	ErrNoChoices = errors.New("model returned no choices")
)

var (
	_ worker.Provider = (*Provider)(nil)
	_ worker.Convo    = (*openAIConvo)(nil)
)

// New returns an adapter serving the configured OpenAI-compatible providers
func New(urls map[api.ProviderID]string) (*Provider, error) {
	models, err := loadModelCapabilities(urls)
	if err != nil {
		return nil, err
	}
	return &Provider{
		urls:   urls,
		models: models,
	}, nil
}

// Supports reports whether this adapter serves the named provider
func (p *Provider) Supports(name api.ProviderID) bool {
	_, ok := p.urls[name]
	return ok || name == Name
}

// Names lists configured providers and OpenAI when its key is set
func (p *Provider) Names() []api.ProviderID {
	var names []api.ProviderID
	for name := range p.urls {
		names = append(names, name)
	}
	if _, configured := p.urls[Name]; !configured && os.Getenv("OPENAI_API_KEY") != "" {
		names = append(names, Name)
	}
	slices.Sort(names)
	return names
}

// Models lists available models for the named provider
func (p *Provider) Models(ctx context.Context, name api.ProviderID) ([]api.AgentModel, error) {
	var models []api.AgentModel
	client := p.openAIClient(name)
	pager := client.Models.ListAutoPaging(ctx)
	now := time.Now()
	for pager.Next() {
		m := pager.Current()
		if !m.ShutdownDate.IsZero() && !now.Before(m.ShutdownDate) {
			continue
		}
		id := api.ModelID(m.ID)
		caps := p.models[worker.ModelKey{
			Provider: name,
			Model:    id,
		}]
		models = append(models, api.AgentModel{
			ID:      id,
			Efforts: caps.Efforts,
		})
	}
	return models, pager.Err()
}

// Open starts a conversation with the task's context and tools
func (p *Provider) Open(ch *worker.Chat) (worker.Convo, error) {
	cfg := ch.Config
	caps := p.models[worker.ModelKey{
		Provider: cfg.Provider,
		Model:    cfg.Model,
	}]
	if caps.Endpoint == responsesEndpoint {
		return newResponsesConvo(p.openAIClient(cfg.Provider), ch), nil
	}
	tools := make([]openai.ChatCompletionToolUnionParam, 0, len(ch.Tools))
	for _, t := range ch.Tools {
		tools = append(tools, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  t.Schema,
		}))
	}
	return &openAIConvo{
		client: p.openAIClient(ch.Config.Provider),
		params: openai.ChatCompletionNewParams{
			Model:               shared.ChatModel(ch.Config.Model),
			MaxCompletionTokens: openai.Int(int64(ch.Config.MaxTokens)),
			ReasoningEffort:     shared.ReasoningEffort(cfg.Effort),
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(ch.System),
				openai.UserMessage(ch.User),
			},
			Tools: tools,
		},
	}, nil
}

// Turn runs one OpenAI-compatible model turn
func (o *openAIConvo) Turn(ctx context.Context) (worker.Turn, error) {
	comp, err := o.client.Chat.Completions.New(ctx, o.params)
	if err != nil {
		return worker.Turn{}, openAIProblem(err)
	}
	res := worker.Turn{
		Cost:              reportedCost(comp.Usage.JSON.ExtraFields["cost"].Raw()),
		InputTokens:       comp.Usage.PromptTokens,
		OutputTokens:      comp.Usage.CompletionTokens,
		CacheInputTokens:  comp.Usage.PromptTokensDetails.CachedTokens,
		CacheOutputTokens: comp.Usage.PromptTokensDetails.CacheWriteTokens,
	}
	if len(comp.Choices) == 0 {
		return res, ErrNoChoices
	}
	msg := comp.Choices[0].Message
	o.params.Messages = append(o.params.Messages, msg.ToParam())
	for _, c := range msg.ToolCalls {
		res.Calls = append(res.Calls, worker.ToolCall{
			ID:   c.ID,
			Name: c.Function.Name,
			Args: json.RawMessage(c.Function.Arguments),
		})
	}
	return res, nil
}

// Answer adds the tool results to the conversation
func (o *openAIConvo) Answer(results []worker.ToolResult) {
	for _, r := range results {
		o.params.Messages = append(o.params.Messages, openai.ToolMessage(r.Content, r.CallID))
	}
}

// Nudge adds a user message to the conversation
func (o *openAIConvo) Nudge(text string) {
	o.params.Messages = append(o.params.Messages, openai.UserMessage(text))
}

func (p *Provider) openAIClient(provider api.ProviderID) openai.Client {
	opts := []option.RequestOption{option.WithAPIKey(os.Getenv(apiKeyEnv(provider)))}
	if base := p.urls[provider]; base != "" {
		opts = append(opts, option.WithBaseURL(base))
	}
	return openai.NewClient(opts...)
}

func openAIProblem(err error) error {
	if providerError, ok := errors.AsType[*openai.Error](err); ok {
		return &api.ProblemDetails{
			Status: providerError.StatusCode,
			Detail: err.Error(),
		}
	}
	return err
}

func reportedCost(raw string) *float64 {
	if raw == "" {
		return nil
	}
	var cost *float64
	if err := json.Unmarshal([]byte(raw), &cost); err != nil {
		return nil
	}
	return cost
}

func apiKeyEnv(provider api.ProviderID) string {
	envUnsafe := regexp.MustCompile(`[^A-Z0-9]+`)
	return envUnsafe.ReplaceAllString(strings.ToUpper(string(provider)), "_") + "_API_KEY"
}
