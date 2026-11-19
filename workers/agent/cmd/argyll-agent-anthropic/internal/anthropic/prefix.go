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
func (p *Provider) selectSystem(ch *worker.Chat, tools []anthropic.ToolUnionParam) string {
	minTokens := p.prefixMin[ch.Config.Model]
	if minTokens <= 0 || ch.CompactSystem == "" {
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
