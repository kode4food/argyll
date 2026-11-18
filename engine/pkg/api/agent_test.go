package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestAgentStepValidate(t *testing.T) {
	assert.NoError(t, newAgentStep(agentConfig()).Validate())
	assert.ErrorIs(t, newAgentStep(nil).Validate(), api.ErrAgentRequired)

	for _, tc := range []struct {
		name string
		cfg  *api.AgentConfig
		err  error
	}{
		{
			name: "empty provider",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.Provider = ""
				return c
			}),
			err: api.ErrAgentProviderEmpty,
		},
		{
			name: "empty model",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.Model = ""
				return c
			}),
			err: api.ErrAgentModelEmpty,
		},
		{
			name: "empty prologue",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.Prologue = ""
				return c
			}),
			err: api.ErrAgentPrologueEmpty,
		},
		{
			name: "no turns",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.MaxTurns = 0
				return c
			}),
			err: api.ErrAgentMaxTurns,
		},
		{
			name: "no tokens",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.MaxTokens = 0
				return c
			}),
			err: api.ErrAgentMaxTokens,
		},
		{
			name: "negative timeout",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.Timeout = -1
				return c
			}),
			err: api.ErrAgentNegativeTimeout,
		},
		{
			name: "empty server",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.MCP = []api.MCPServer{{}}
				return c
			}),
			err: api.ErrMCPServerEmpty,
		},
		{
			name: "duplicate server",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.MCP = []api.MCPServer{{Server: "crm"}, {Server: "crm"}}
				return c
			}),
			err: api.ErrMCPServerDuplicate,
		},
		{
			name: "empty tool",
			cfg: withAgent(func(c api.AgentConfig) api.AgentConfig {
				c.MCP = []api.MCPServer{
					{
						Server: "crm",
						Tools:  []string{""},
					},
				}
				return c
			}),
			err: api.ErrMCPToolEmpty,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, newAgentStep(tc.cfg).Validate(), tc.err)
		})
	}
}

func TestAgentOtherConfigs(t *testing.T) {
	http := &api.HTTPConfig{Invoke: api.HTTPAction{Endpoint: "http://x"}}
	script := &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Source:   "1",
	}
	flow := &api.FlowConfig{Goals: api.Goals{Steps: []api.StepID{"x"}}}

	for _, tc := range []struct {
		step *api.Step
		err  error
	}{
		{
			step: &api.Step{
				ID:    "agent",
				Name:  "Agent",
				Type:  api.StepTypeAgent,
				Agent: agentConfig(),
				HTTP:  http,
			},
			err: api.ErrHTTPNotAllowed,
		},
		{
			step: &api.Step{
				ID:     "agent",
				Name:   "Agent",
				Type:   api.StepTypeAgent,
				Agent:  agentConfig(),
				Script: script,
			},
			err: api.ErrScriptNotAllowed,
		},
		{
			step: &api.Step{
				ID:    "agent",
				Name:  "Agent",
				Type:  api.StepTypeAgent,
				Agent: agentConfig(),
				Flow:  flow,
			},
			err: api.ErrFlowNotAllowed,
		},
		{
			step: &api.Step{
				ID:    "svc",
				Name:  "Service",
				Type:  api.StepTypeService,
				HTTP:  http,
				Agent: agentConfig(),
			},
			err: api.ErrAgentNotAllowed,
		},
		{
			step: &api.Step{
				ID:     "scr",
				Name:   "Script",
				Type:   api.StepTypeScript,
				Script: script,
				Agent:  agentConfig(),
			},
			err: api.ErrAgentNotAllowed,
		},
		{
			step: &api.Step{
				ID:    "flw",
				Name:  "Flow",
				Type:  api.StepTypeFlow,
				Flow:  flow,
				Agent: agentConfig(),
			},
			err: api.ErrAgentNotAllowed,
		},
	} {
		assert.ErrorIs(t, tc.step.Validate(), tc.err)
	}
}

func TestAgentStepEqual(t *testing.T) {
	a := newAgentStep(agentConfig())
	assert.True(t, a.Equal(newAgentStep(agentConfig())))
	assert.False(t, a.Equal(newAgentStep(nil)))
	assert.True(t, (*api.AgentConfig)(nil).Equal(nil))

	for _, cfg := range []*api.AgentConfig{
		withAgent(func(c api.AgentConfig) api.AgentConfig {
			c.MCP = []api.MCPServer{
				{
					Server: "crm",
					Tools:  []string{"other"},
				},
			}
			return c
		}),
		withAgent(func(c api.AgentConfig) api.AgentConfig {
			c.Model = "other"
			return c
		}),
		withAgent(func(c api.AgentConfig) api.AgentConfig {
			c.Timeout = 1
			return c
		}),
	} {
		assert.False(t, a.Equal(newAgentStep(cfg)))
	}
}

func TestAgentStepHashKey(t *testing.T) {
	a, err := newAgentStep(agentConfig()).HashKey()
	assert.NoError(t, err)

	b, err := newAgentStep(withAgent(func(c api.AgentConfig) api.AgentConfig {
		c.Prologue = "changed"
		return c
	})).HashKey()
	assert.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestAgentStepCopy(t *testing.T) {
	st := newAgentStep(agentConfig())
	assert.Same(t, st.Agent, st.Copy().Agent)
}

func TestAgentMCPToolSelection(t *testing.T) {
	all := api.MCPServer{Server: "tools"}
	var none api.MCPServer
	assert.NoError(t, json.Unmarshal([]byte(`{"server":"tools","tools":[]}`), &none))
	assert.False(t, all.Equal(none))
	assert.True(t, none.Equal(none))

	data, err := json.Marshal(none)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"server":"tools","tools":[]}`, string(data))
	data, err = json.Marshal(all)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"server":"tools"}`, string(data))
}

func TestAgentMappingCollision(t *testing.T) {
	st := newAgentStep(agentConfig())
	st.Attributes["score"] = &api.AttributeSpec{Role: api.RoleOutput}
	st.Attributes["rating"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Output: &api.OutputConfig{
			Mapping: &api.MappingConfig{Name: "score"},
		},
	}
	assert.ErrorIs(t, st.Validate(), api.ErrDuplicateInnerName)
}

func newAgentStep(cfg *api.AgentConfig) *api.Step {
	return &api.Step{
		ID:    "agent",
		Name:  "Agent",
		Type:  api.StepTypeAgent,
		Agent: cfg,
		Attributes: api.AttributeSpecs{
			"order": {Role: api.RoleRequired},
			"score": {Role: api.RoleOutput},
		},
	}
}

// withAgent returns the standard agent config as the given function rewrites it
func withAgent(fn func(api.AgentConfig) api.AgentConfig) *api.AgentConfig {
	res := fn(*agentConfig())
	return &res
}

func agentConfig() *api.AgentConfig {
	return &api.AgentConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-5-5",
		Prologue: "# Score\n\nScore the `order`",
		MCP: []api.MCPServer{
			{
				Server: "crm",
				Tools:  []string{"lookup"},
			},
		},
		MaxTurns:  4,
		MaxTokens: 1024,
	}
}
