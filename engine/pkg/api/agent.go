package api

import (
	"errors"
	"fmt"
	"slices"
)

type (
	// AgentConfig configures agent-based step execution, run by the worker serving its provider
	AgentConfig struct {
		MCP       []MCPServer `json:"mcp,omitempty"`
		Provider  string      `json:"provider"`
		Model     string      `json:"model"`
		Prologue  string      `json:"prologue"`
		MaxTurns  int         `json:"max_turns"`
		MaxTokens int         `json:"max_tokens"`
		Timeout   int64       `json:"timeout,omitempty"`
	}

	// AgentRequest is the envelope the engine sends a worker to run one attempt of agent work
	AgentRequest struct {
		Step    *Step `json:"step"`
		Inputs  Args  `json:"inputs"`
		Timeout int64 `json:"timeout"`
	}

	// AgentOutcome is the callback envelope for one Agent attempt
	AgentOutcome struct {
		Outputs Args            `json:"outputs,omitzero"`
		Problem *ProblemDetails `json:"problem,omitempty"`
		Usage   Usage           `json:"usage,omitempty"`
	}

	// AgentCapabilities lists the providers available for Agent Steps
	AgentCapabilities struct {
		Providers []AgentProvider `json:"providers"`
	}

	// AgentProvider lists a provider's models and the MCP servers on its worker
	AgentProvider struct {
		Provider string      `json:"provider"`
		Models   []string    `json:"models"`
		MCP      []MCPServer `json:"mcp"`
	}

	// MCPServer names an MCP server an agent may use, optionally limited to some of its tools
	MCPServer struct {
		Tools  []string `json:"tools,omitzero"`
		Server string   `json:"server"`
	}
)

var (
	ErrAgentRequired        = errors.New("agent required")
	ErrAgentNotAllowed      = errors.New("agent not allowed for step type")
	ErrAgentProviderEmpty   = errors.New("agent provider empty")
	ErrAgentModelEmpty      = errors.New("agent model empty")
	ErrAgentPrologueEmpty   = errors.New("agent prologue empty")
	ErrAgentMaxTurns        = errors.New("agent max_turns must be positive")
	ErrAgentMaxTokens       = errors.New("agent max_tokens must be positive")
	ErrAgentNegativeTimeout = errors.New("agent timeout cannot be negative")
	ErrMCPServerEmpty       = errors.New("MCP server name empty")
	ErrMCPServerDuplicate   = errors.New("duplicate MCP server")
	ErrMCPToolEmpty         = errors.New("MCP tool name empty")
)

// Equal returns true if two agent configs are equal
func (c *AgentConfig) Equal(other *AgentConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	sameModel := c.Provider == other.Provider && c.Model == other.Model &&
		c.Prologue == other.Prologue
	sameLimits := c.MaxTurns == other.MaxTurns && c.MaxTokens == other.MaxTokens &&
		c.Timeout == other.Timeout
	return sameModel && sameLimits && slices.EqualFunc(c.MCP, other.MCP, MCPServer.Equal)
}

// Equal returns true if two MCP server selections are equal
func (m MCPServer) Equal(other MCPServer) bool {
	return m.Server == other.Server && (m.Tools == nil) == (other.Tools == nil) &&
		slices.Equal(m.Tools, other.Tools)
}

func (s *Step) validateAgentConfig() error {
	if s.Agent == nil {
		return ErrAgentRequired
	}
	if s.HTTP != nil {
		return ErrHTTPNotAllowed
	}
	if s.Script != nil {
		return ErrScriptNotAllowed
	}
	if s.Flow != nil {
		return ErrFlowNotAllowed
	}
	return s.Agent.validate()
}

func (c *AgentConfig) validate() error {
	switch {
	case c.Provider == "":
		return ErrAgentProviderEmpty
	case c.Model == "":
		return ErrAgentModelEmpty
	case c.Prologue == "":
		return ErrAgentPrologueEmpty
	case c.MaxTurns <= 0:
		return ErrAgentMaxTurns
	case c.MaxTokens <= 0:
		return ErrAgentMaxTokens
	case c.Timeout < 0:
		return ErrAgentNegativeTimeout
	}
	return validateMCPServers(c.MCP)
}

func validateMCPServers(servers []MCPServer) error {
	seen := map[string]bool{}
	for _, m := range servers {
		if m.Server == "" {
			return ErrMCPServerEmpty
		}
		if seen[m.Server] {
			return fmt.Errorf("%w: %s", ErrMCPServerDuplicate, m.Server)
		}
		seen[m.Server] = true
		if slices.Contains(m.Tools, "") {
			return fmt.Errorf("%w: %s", ErrMCPToolEmpty, m.Server)
		}
	}
	return nil
}
