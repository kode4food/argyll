package contract_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestAgentChatRequestValidate(t *testing.T) {
	user := contract.ChatMessage{
		Role:    contract.ChatRoleUser,
		Content: "hi",
	}
	reply := contract.ChatMessage{
		Role:    contract.ChatRoleAssistant,
		Content: "hello",
	}
	for _, tc := range []struct {
		name     string
		provider api.ProviderID
		model    api.ModelID
		messages []contract.ChatMessage
		err      error
	}{
		{
			name:     "valid",
			provider: "anthropic",
			model:    "m",
			messages: []contract.ChatMessage{user, reply, user},
		},
		{
			name:     "empty provider",
			model:    "m",
			messages: []contract.ChatMessage{user},
			err:      api.ErrAgentProviderEmpty,
		},
		{
			name:     "empty model",
			provider: "anthropic",
			messages: []contract.ChatMessage{user},
			err:      api.ErrAgentModelEmpty,
		},
		{
			name:     "no messages",
			provider: "anthropic",
			model:    "m",
			err:      contract.ErrChatMessagesEmpty,
		},
		{
			name:     "last not user",
			provider: "anthropic",
			model:    "m",
			messages: []contract.ChatMessage{user, reply},
			err:      contract.ErrChatLastNotUser,
		},
		{
			name:     "invalid role",
			provider: "anthropic",
			model:    "m",
			messages: []contract.ChatMessage{{Role: "system", Content: "x"}, user},
			err:      contract.ErrChatRoleInvalid,
		},
		{
			name:     "empty content",
			provider: "anthropic",
			model:    "m",
			messages: []contract.ChatMessage{{Role: contract.ChatRoleUser}},
			err:      contract.ErrChatContentEmpty,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &contract.AgentChatRequest{
				Provider: tc.provider,
				Model:    tc.model,
				Messages: tc.messages,
			}
			if tc.err == nil {
				assert.NoError(t, req.Validate())
				return
			}
			assert.ErrorIs(t, req.Validate(), tc.err)
		})
	}
}

func TestChatActionValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action contract.ChatAction
		err    error
	}{
		{
			name:   "overview",
			action: contract.ChatAction{Type: contract.ChatActionOpenOverview},
		},
		{
			name:   "new step",
			action: contract.ChatAction{Type: contract.ChatActionNewStep},
		},
		{
			name:   "space list",
			action: contract.ChatAction{Type: contract.ChatActionOpenSpace},
		},
		{
			name:   "feed list",
			action: contract.ChatAction{Type: contract.ChatActionOpenFeed},
		},
		{
			name: "step draft",
			action: contract.ChatAction{
				Type:  contract.ChatActionNewStep,
				Draft: json.RawMessage(`{"name": "Score", "type": "script"}`),
			},
		},
		{
			name: "space draft",
			action: contract.ChatAction{
				Type:  contract.ChatActionNewSpace,
				Draft: json.RawMessage(`{"qbe": [["domain:risk"]]}`),
			},
		},
		{
			name: "feed draft",
			action: contract.ChatAction{
				Type:  contract.ChatActionNewFeed,
				Draft: json.RawMessage(`{"value": "0", "attributes": ["total"]}`),
			},
		},
		{
			name:   "empty feed",
			action: contract.ChatAction{Type: contract.ChatActionNewFeed},
		},
		{
			name: "misshapen draft",
			action: contract.ChatAction{
				Type:  contract.ChatActionNewSpace,
				Draft: json.RawMessage(`{"name": 5}`),
			},
			err: contract.ErrChatActionDraft,
		},
		{
			name: "draft not an object",
			action: contract.ChatAction{
				Type:  contract.ChatActionNewStep,
				Draft: json.RawMessage(`"score"`),
			},
			err: contract.ErrChatActionDraft,
		},
		{
			name: "flow",
			action: contract.ChatAction{
				Type:   contract.ChatActionOpenFlow,
				FlowID: "f",
			},
		},
		{
			name: "goals",
			action: contract.ChatAction{
				Type:  contract.ChatActionSelectGoals,
				Goals: &api.Goals{Steps: []api.StepID{"s"}},
			},
		},
		{
			name: "step",
			action: contract.ChatAction{
				Type:   contract.ChatActionOpenStep,
				StepID: "s",
			},
		},
		{
			name:   "flow without id",
			action: contract.ChatAction{Type: contract.ChatActionOpenFlow},
			err:    contract.ErrChatActionTarget,
		},
		{
			name: "empty goals",
			action: contract.ChatAction{
				Type:  contract.ChatActionSelectGoals,
				Goals: &api.Goals{},
			},
			err: contract.ErrChatActionTarget,
		},
		{
			name:   "step without id",
			action: contract.ChatAction{Type: contract.ChatActionOpenStep},
			err:    contract.ErrChatActionTarget,
		},
		{
			name:   "unknown type",
			action: contract.ChatAction{Type: "start_flow"},
			err:    contract.ErrChatActionType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				assert.NoError(t, tc.action.Validate())
				return
			}
			assert.ErrorIs(t, tc.action.Validate(), tc.err)
		})
	}
}
