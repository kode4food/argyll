package anthropic

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

type (
	// Provider runs agent work through the Anthropic API
	Provider struct {
		url       string
		prefixMin map[api.ModelID]int
	}

	convo struct {
		client anthropic.Client
		params anthropic.MessageNewParams
	}
)

// Name identifies the Anthropic provider
const Name api.ProviderID = "anthropic"

var (
	_ worker.Provider = (*Provider)(nil)
	_ worker.Convo    = (*convo)(nil)
)

// New returns an Anthropic provider using the configured base URL
func New(urls map[api.ProviderID]string) (*Provider, error) {
	thresholds, err := loadPrefixThresholds()
	if err != nil {
		return nil, err
	}
	return &Provider{
		url:       urls[Name],
		prefixMin: thresholds,
	}, nil
}

// Supports reports whether this adapter serves the named provider
func (*Provider) Supports(name api.ProviderID) bool {
	return name == Name
}

// Names lists Anthropic when a key or base URL is configured
func (p *Provider) Names() []api.ProviderID {
	if p.url != "" || os.Getenv("ANTHROPIC_API_KEY") != "" {
		return []api.ProviderID{Name}
	}
	return nil
}

// Models lists available Anthropic models
func (p *Provider) Models(ctx context.Context, _ api.ProviderID) ([]api.AgentModel, error) {
	var models []api.AgentModel
	client := p.client(os.Getenv("ANTHROPIC_API_KEY"))
	pager := client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	for pager.Next() {
		m := pager.Current()
		if m.Lifecycle == anthropic.ModelInfoLifecycleRetired {
			continue
		}
		models = append(models, api.AgentModel{
			ID:      api.ModelID(m.ID),
			Efforts: effortLevels(m.Capabilities.Effort),
		})
	}
	return models, pager.Err()
}

// Open starts an Anthropic conversation with the task's context and tools
func (p *Provider) Open(ch *worker.Chat) (worker.Convo, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("%w: %s", worker.ErrMissingAPIKey, "ANTHROPIC_API_KEY")
	}
	tools := make([]anthropic.ToolUnionParam, 0, len(ch.Tools))
	for _, t := range ch.Tools {
		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: toolSchema(t.Schema),
			},
		})
	}
	return &convo{
		client: p.client(key),
		params: anthropic.MessageNewParams{
			Model:     anthropic.Model(ch.Config.Model),
			MaxTokens: int64(ch.Config.MaxTokens),
			System: []anthropic.TextBlockParam{
				{
					Text:         p.selectSystem(ch, tools),
					CacheControl: anthropic.NewCacheControlEphemeralParam(),
				},
			},
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(ch.User)),
			},
			Tools: tools,
			OutputConfig: anthropic.OutputConfigParam{
				Effort: anthropic.OutputConfigEffort(ch.Config.Effort),
			},
		},
	}, nil
}

// Turn runs one Anthropic model turn
func (c *convo) Turn(ctx context.Context) (worker.Turn, error) {
	msg, err := c.client.Messages.New(ctx, c.params)
	if err != nil {
		if providerError, ok := errors.AsType[*anthropic.Error](err); ok {
			return worker.Turn{}, &api.ProblemDetails{
				Status: providerError.StatusCode,
				Detail: err.Error(),
			}
		}
		return worker.Turn{}, err
	}
	c.params.Messages = append(c.params.Messages, msg.ToParam())
	usage := msg.Usage
	input := usage.CacheReadInputTokens
	output := usage.CacheCreationInputTokens
	res := worker.Turn{
		InputTokens:       usage.InputTokens + input + output,
		OutputTokens:      usage.OutputTokens,
		CacheInputTokens:  input,
		CacheOutputTokens: output,
	}
	for _, block := range msg.Content {
		use, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}
		res.Calls = append(res.Calls, worker.ToolCall{
			ID:   use.ID,
			Name: use.Name,
			Args: use.Input,
		})
	}
	return res, nil
}

// Answer adds tool replies to the conversation
func (c *convo) Answer(results []worker.ToolResult) {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(results))
	for _, r := range results {
		blocks = append(blocks, anthropic.NewToolResultBlock(r.CallID, r.Content, false))
	}
	c.params.Messages = append(c.params.Messages, anthropic.NewUserMessage(blocks...))
}

// Nudge adds an instruction to the conversation
func (c *convo) Nudge(text string) {
	c.params.Messages = append(
		c.params.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(text)),
	)
}

func (p *Provider) client(key string) anthropic.Client {
	opts := []option.RequestOption{option.WithAPIKey(key)}
	if p.url != "" {
		opts = append(opts, option.WithBaseURL(p.url))
	}
	return anthropic.NewClient(opts...)
}

func effortLevels(effort anthropic.EffortCapability) []api.Effort {
	if !effort.Supported {
		return nil
	}
	var levels []api.Effort
	for _, level := range []struct {
		name      api.Effort
		supported bool
	}{
		{name: api.Effort(anthropic.OutputConfigEffortLow), supported: effort.Low.Supported},
		{name: api.Effort(anthropic.OutputConfigEffortMedium), supported: effort.Medium.Supported},
		{name: api.Effort(anthropic.OutputConfigEffortHigh), supported: effort.High.Supported},
		{name: api.Effort(anthropic.OutputConfigEffortXhigh), supported: effort.Xhigh.Supported},
		{name: api.Effort(anthropic.OutputConfigEffortMax), supported: effort.Max.Supported},
	} {
		if level.supported {
			levels = append(levels, level.name)
		}
	}
	return levels
}

func toolSchema(schema map[string]any) anthropic.ToolInputSchemaParam {
	res := anthropic.ToolInputSchemaParam{
		Properties:  schema["properties"],
		ExtraFields: map[string]any{},
	}
	for k, v := range schema {
		switch k {
		case "type", "properties":
		case "required":
			res.Required = requiredNames(v)
		default:
			res.ExtraFields[k] = v
		}
	}
	return res
}

func requiredNames(v any) []string {
	names, _ := v.([]any)
	res := make([]string, 0, len(names))
	for _, n := range names {
		if s, ok := n.(string); ok {
			res = append(res, s)
		}
	}
	return res
}
