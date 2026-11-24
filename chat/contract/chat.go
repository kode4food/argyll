// Package contract defines the HTTP messages shared by chat and its workers
package contract

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// AgentChatRequest asks an agent to answer the latest message of a conversation
	AgentChatRequest struct {
		Provider api.ProviderID `json:"provider"`
		Model    api.ModelID    `json:"model"`
		Messages []ChatMessage  `json:"messages"`
	}

	// AgentChatReply carries an agent's answer and the usage it took
	AgentChatReply struct {
		Reply string    `json:"reply"`
		Usage api.Usage `json:"usage"`
	}

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
)

var (
	ErrChatMessagesEmpty = errors.New("chat messages empty")
	ErrChatRoleInvalid   = errors.New("chat role invalid")
	ErrChatContentEmpty  = errors.New("chat message content empty")
	ErrChatLastNotUser   = errors.New("last chat message must come from the user")
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
