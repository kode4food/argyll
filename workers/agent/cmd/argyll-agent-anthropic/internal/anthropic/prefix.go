package anthropic

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const (
	systemBytesPerToken = 4
	toolBytesPerToken   = 3
	toolPromptTokens    = 286
	tokenEstimateMargin = 32
)

// selectSystem estimates cache eligibility from the stable text and tool schemas
func selectSystem(ch *worker.Chat, tools []anthropic.ToolUnionParam) string {
	minTokens := cacheMinimum(ch.Config.Model)
	if minTokens == 0 || ch.CompactSystem == "" {
		return ch.System
	}
	b, err := json.Marshal(tools)
	if err != nil {
		return ch.System
	}
	toolTokens := 0
	if len(tools) > 0 {
		// Claude also adds its own instructions for client tools
		toolTokens = toolPromptTokens + len(b)/toolBytesPerToken
	}
	compactTokens := toolTokens + len(ch.CompactSystem)/systemBytesPerToken
	expandedTokens := toolTokens + len(ch.System)/systemBytesPerToken
	if compactTokens >= minTokens+tokenEstimateMargin ||
		expandedTokens < minTokens-tokenEstimateMargin {
		return ch.CompactSystem
	}
	return ch.System
}

func cacheMinimum(model string) int {
	switch model {
	case "claude-fable-5-2", "claude-fable-5-1", "claude-fable-5", "claude-opus-5-5",
		"claude-opus-5", "claude-sonnet-5-5", "claude-haiku-5-5":
		return 512
	case "claude-opus-4-8", "claude-sonnet-5", "claude-sonnet-4-6", "claude-sonnet-4-5",
		"claude-sonnet-4-5-20250929":
		return 1024
	case "claude-opus-4-7":
		return 2048
	case "claude-opus-4-6", "claude-opus-4-5", "claude-opus-4-5-20251101",
		"claude-haiku-4-5", "claude-haiku-4-5-20251001":
		return 4096
	default:
		return 0
	}
}
