package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

// convo resends the whole conversation each turn, so nothing is stored by the provider
type convo struct {
	client  *genai.Client
	request interactions.CreateModelInteraction
	steps   []interactions.Step
	names   map[string]string
}

var (
	ErrNoInteraction = errors.New("model returned no interaction")
)

var _ worker.Convo = (*convo)(nil)

// Turn runs one Gemini model turn
func (c *convo) Turn(ctx context.Context) (worker.Turn, error) {
	input := interactions.NewInteractionsInput(c.steps)
	c.request.Input = &input
	res, err := c.client.Interactions.Create(ctx, operations.CreateInteractionRequest{
		Body: operations.NewCreateInteractionRequestBody(c.request),
	})
	if err != nil {
		return worker.Turn{}, problem(err)
	}
	out := res.GetInteraction()
	if out == nil {
		return worker.Turn{}, ErrNoInteraction
	}
	turn := usage(out.Usage)
	switch out.Status {
	case interactions.InteractionStatusCompleted, interactions.InteractionStatusRequiresAction:
	default:
		return turn, fmt.Errorf(
			"%w: gemini: %s%s", worker.ErrModelStopped, out.Status, errorText(out.Errors),
		)
	}
	// Thought steps carry signatures the model needs back exactly as it sent them
	c.steps = append(c.steps, out.Steps...)
	for _, st := range out.Steps {
		call := st.FunctionCallStep
		if call == nil {
			continue
		}
		args, err := json.Marshal(call.Arguments)
		if err != nil {
			return turn, err
		}
		c.names[call.ID] = call.Name
		turn.Calls = append(turn.Calls, worker.ToolCall{
			ID:   call.ID,
			Name: call.Name,
			Args: args,
		})
	}
	return turn, nil
}

// Answer adds tool replies to the conversation
func (c *convo) Answer(results []worker.ToolResult) {
	for _, r := range results {
		c.steps = append(c.steps, interactions.NewStep(interactions.FunctionResultStep{
			CallID: r.CallID,
			Name:   new(c.names[r.CallID]),
			Result: interactions.NewFunctionResultStepResultUnion(r.Content),
		}))
	}
}

// Nudge adds an instruction to the conversation
func (c *convo) Nudge(text string) {
	c.steps = append(c.steps, userText(text))
}

func newConvo(client *genai.Client, ch *worker.Chat) *convo {
	cfg := ch.Config
	tools := make([]interactions.Tool, 0, len(ch.Tools))
	for _, t := range ch.Tools {
		tools = append(tools, interactions.NewTool(interactions.Function{
			Name:        new(t.Name),
			Description: new(t.Description),
			Parameters:  t.Schema,
		}))
	}
	gen := &interactions.GenerationConfig{
		MaxOutputTokens: new(cfg.MaxTokens),
	}
	if cfg.Effort != "" {
		gen.ThinkingLevel = new(interactions.ThinkingLevel(cfg.Effort))
	}
	return &convo{
		client: client,
		request: interactions.CreateModelInteraction{
			Model:             interactions.Model(cfg.Model),
			Store:             new(false),
			SystemInstruction: new(ch.System),
			Tools:             tools,
			GenerationConfig:  gen,
		},
		steps: []interactions.Step{userText(ch.User)},
		names: map[string]string{},
	}
}

// usage counts thinking as output, since Gemini bills it at the output rate
func usage(u *interactions.Usage) worker.Turn {
	if u == nil {
		return worker.Turn{}
	}
	return worker.Turn{
		InputTokens:      count(u.TotalInputTokens) + count(u.TotalToolUseTokens),
		OutputTokens:     count(u.TotalOutputTokens) + count(u.TotalThoughtTokens),
		CacheInputTokens: count(u.TotalCachedTokens),
	}
}

func count(n *int) int64 {
	if n == nil {
		return 0
	}
	return int64(*n)
}

func userText(text string) interactions.Step {
	return interactions.NewStep(interactions.UserInputStep{
		Content: []interactions.Content{
			interactions.NewContent(interactions.TextContent{Text: text}),
		},
	})
}
