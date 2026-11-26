// Package contract defines the HTTP messages shared by chat and its workers
package contract

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// AgentChatRequest asks an agent to answer the latest message of a conversation
	AgentChatRequest struct {
		View     *ChatView      `json:"view,omitempty"`
		Provider api.ProviderID `json:"provider"`
		Model    api.ModelID    `json:"model"`
		Messages []ChatMessage  `json:"messages"`
	}

	// ChatView describes what the web UI shows while the user chats
	ChatView struct {
		Goals   *api.Goals   `json:"goals,omitempty"`
		Init    api.InitArgs `json:"init,omitempty"`
		SpaceID api.SpaceID  `json:"space_id,omitempty"`
		FlowID  api.FlowID   `json:"flow_id,omitempty"`
		Locale  string       `json:"locale,omitempty"`
	}

	// AgentChatReply carries an agent's answer and the usage it took
	AgentChatReply struct {
		Action *ChatAction `json:"action,omitempty"`
		Reply  string      `json:"reply"`
		Usage  api.Usage   `json:"usage"`
	}

	// ChatAction takes the web UI to a view or editor
	ChatAction struct {
		Goals   *api.Goals      `json:"goals,omitempty"`
		Draft   json.RawMessage `json:"draft,omitempty"`
		Type    ChatActionType  `json:"type"`
		SpaceID api.SpaceID     `json:"space_id,omitempty"`
		FlowID  api.FlowID      `json:"flow_id,omitempty"`
		StepID  api.StepID      `json:"step_id,omitempty"`
		FeedID  api.FeedID      `json:"feed_id,omitempty"`
	}

	// ChatActionType names where a ChatAction takes the web UI
	ChatActionType string

	// ChatMessage is one message of a conversation
	ChatMessage struct {
		Role    ChatRole `json:"role"`
		Content string   `json:"content"`
	}

	// ChatRole identifies who wrote a chat message
	ChatRole string
)

const (
	Path                       = "/chat"
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"

	ChatActionOpenOverview ChatActionType = "open_overview"
	ChatActionOpenFlow     ChatActionType = "open_flow"
	ChatActionSelectGoals  ChatActionType = "select_goals"
	ChatActionOpenStep     ChatActionType = "open_step"
	ChatActionNewStep      ChatActionType = "new_step"
	ChatActionOpenSpace    ChatActionType = "open_space"
	ChatActionNewSpace     ChatActionType = "new_space"
	ChatActionOpenFeed     ChatActionType = "open_feed"
	ChatActionNewFeed      ChatActionType = "new_feed"
)

var (
	ErrChatMessagesEmpty = errors.New("chat messages empty")
	ErrChatRoleInvalid   = errors.New("chat role invalid")
	ErrChatContentEmpty  = errors.New("chat message content empty")
	ErrChatLastNotUser   = errors.New("last chat message must come from the user")
	ErrChatActionType    = errors.New("chat action type unknown")
	ErrChatActionTarget  = errors.New("chat action missing its target")
	ErrChatActionDraft   = errors.New("chat action draft invalid")
)

// Validate checks that a chat request names a model and ends with a user message
func (r *AgentChatRequest) Validate() error {
	switch {
	case r.Provider == "":
		return api.ErrAgentProviderEmpty
	case r.Model == "":
		return api.ErrAgentModelEmpty
	case len(r.Messages) == 0:
		return ErrChatMessagesEmpty
	case r.Messages[len(r.Messages)-1].Role != ChatRoleUser:
		return ErrChatLastNotUser
	}
	for _, m := range r.Messages {
		if m.Role != ChatRoleUser && m.Role != ChatRoleAssistant {
			return ErrChatRoleInvalid
		}
		if m.Content == "" {
			return ErrChatContentEmpty
		}
	}
	return nil
}

// Validate checks an action's type, target, and draft
func (a *ChatAction) Validate() error {
	switch a.Type {
	case ChatActionOpenOverview, ChatActionOpenSpace, ChatActionOpenFeed:
		return nil
	case ChatActionOpenFlow:
		return requireTarget(a.FlowID != "", "flow_id")
	case ChatActionSelectGoals:
		return requireTarget(a.Goals != nil && len(a.Goals.Steps) > 0, "goals")
	case ChatActionOpenStep:
		return requireTarget(a.StepID != "", "step_id")
	case ChatActionNewStep:
		return checkDraft[api.Step](a.Draft)
	case ChatActionNewSpace:
		return checkDraft[api.Space](a.Draft)
	case ChatActionNewFeed:
		return checkDraft[api.Feed](a.Draft)
	}
	return fmt.Errorf("%w: %s", ErrChatActionType, a.Type)
}

func requireTarget(ok bool, field string) error {
	if ok {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrChatActionTarget, field)
}

func checkDraft[T any](draft json.RawMessage) error {
	if len(draft) == 0 {
		return nil
	}
	var v T
	if err := json.Unmarshal(draft, &v); err != nil {
		return fmt.Errorf("%w: %w", ErrChatActionDraft, err)
	}
	return nil
}
