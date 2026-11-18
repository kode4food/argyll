package openai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	// Provider runs agent work through OpenAI-compatible APIs
	Provider struct {
		urls map[string]string
	}

	openAIConvo struct {
		client openai.Client
		params openai.ChatCompletionNewParams
	}

	openAIUsage struct {
		Cost                *float64 `json:"cost"`
		PromptTokensDetails struct {
			CacheOutputTokens int64 `json:"cache_write_tokens"`
		} `json:"prompt_tokens_details"`
	}
)

// Name identifies the default OpenAI provider
const Name = "openai"

var (
	ErrNoChoices = errors.New("model returned no choices")
)

var _ worker.Convo = (*openAIConvo)(nil)

// New returns an adapter serving the configured OpenAI-compatible providers
func New(urls map[string]string) *Provider {
	return &Provider{urls: urls}
}

// Supports reports whether this adapter serves the named provider
func (p *Provider) Supports(name string) bool {
	if name == "anthropic" {
		return false
	}
	_, ok := p.urls[name]
	return ok || name == Name
}

// Names lists configured providers and OpenAI when its key is set
func (p *Provider) Names() []string {
	var names []string
	for name := range p.urls {
		if p.Supports(name) {
			names = append(names, name)
		}
	}
	if _, configured := p.urls[Name]; !configured && os.Getenv("OPENAI_API_KEY") != "" {
		names = append(names, Name)
	}
	slices.Sort(names)
	return names
}

// Models lists available models for the named provider
func (p *Provider) Models(ctx context.Context, name string) ([]string, error) {
	var models []string
	client := p.openAIClient(name)
	pager := client.Models.ListAutoPaging(ctx)
	for pager.Next() {
		models = append(models, pager.Current().ID)
	}
	return models, pager.Err()
}

// Open starts a conversation with the task's context and tools
func (p *Provider) Open(ch *worker.Chat) (worker.Convo, error) {
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
			Model:               ch.Config.Model,
			MaxCompletionTokens: openai.Int(int64(ch.Config.MaxTokens)),
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
		var providerError *openai.Error
		if errors.As(err, &providerError) {
			return worker.Turn{}, &api.ProblemDetails{
				Status: providerError.StatusCode,
				Detail: err.Error(),
			}
		}
		return worker.Turn{}, err
	}
	var usage openAIUsage
	if err := json.Unmarshal([]byte(comp.Usage.RawJSON()), &usage); err != nil {
		usage = openAIUsage{}
	}
	res := worker.Turn{
		Cost:              usage.Cost,
		InputTokens:       comp.Usage.PromptTokens,
		OutputTokens:      comp.Usage.CompletionTokens,
		CacheInputTokens:  comp.Usage.PromptTokensDetails.CachedTokens,
		CacheOutputTokens: usage.PromptTokensDetails.CacheOutputTokens,
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

func (p *Provider) openAIClient(provider string) openai.Client {
	opts := []option.RequestOption{option.WithAPIKey(os.Getenv(apiKeyEnv(provider)))}
	if base := p.urls[provider]; base != "" {
		opts = append(opts, option.WithBaseURL(base))
	}
	return openai.NewClient(opts...)
}

func apiKeyEnv(provider string) string {
	envUnsafe := regexp.MustCompile(`[^A-Z0-9]+`)
	return envUnsafe.ReplaceAllString(strings.ToUpper(provider), "_") + "_API_KEY"
}
