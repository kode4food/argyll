package worker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Provider connects the shared worker to a model API
	Provider interface {
		Supports(provider string) bool
		Names() []string
		Models(ctx context.Context, provider string) ([]string, error)
		Open(*Chat) (Convo, error)
	}

	// Convo is one attempt's exchange with a model
	Convo interface {
		Turn(context.Context) (Turn, error)
		Answer([]ToolResult)
		Nudge(text string)
	}

	// Turn reports one model response and its usage
	Turn struct {
		Calls             []ToolCall
		Cost              *float64
		InputTokens       int64
		OutputTokens      int64
		CacheInputTokens  int64
		CacheOutputTokens int64
	}

	// ToolCall is a model's request to invoke a tool
	ToolCall struct {
		ID   string
		Name string
		Args json.RawMessage
	}

	// ToolResult carries a tool's reply to the model
	ToolResult struct {
		CallID  string
		Content string
	}

	// Chat supplies the initial model context and available tools
	Chat struct {
		Config        *api.AgentConfig
		System        string
		CompactSystem string
		User          string
		Tools         []ToolSpec
	}

	// ToolSpec describes a tool available to a model
	ToolSpec struct {
		Name        string
		Description string
		Schema      map[string]any
	}
)

var (
	ErrMissingAPIKey = errors.New("provider API key not set")
)
