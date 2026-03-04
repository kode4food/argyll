package api

import "encoding/json"

type (
	// WebSocketEvent is an event sent to WebSocket clients
	WebSocketEvent struct {
		Type           EventType       `json:"type"`
		Data           json.RawMessage `json:"data"`
		AggregateID    []string        `json:"id"`
		SubscriptionID string          `json:"sub_id,omitempty"`
		Timestamp      int64           `json:"timestamp"`
		Sequence       int64           `json:"sequence"`
	}

	// SubscribeRequest is sent by clients to subscribe to events
	SubscribeRequest struct {
		Type string             `json:"type"`
		Data ClientSubscription `json:"data"`
	}

	// ClientSubscription configures which events a WebSocket client receives
	ClientSubscription struct {
		SubscriptionID string      `json:"sub_id,omitempty"`
		AggregateIDs   [][]string  `json:"aggregate_ids,omitempty"`
		IncludeState   bool        `json:"include_state,omitempty"`
		EventTypes     []EventType `json:"event_types,omitempty"`
	}

	// UnsubscribeRequest is sent by clients to remove an active subscription
	UnsubscribeRequest struct {
		Type string               `json:"type"`
		Data ClientUnsubscription `json:"data"`
	}

	// ClientUnsubscription removes a previously registered subscription
	ClientUnsubscription struct {
		SubscriptionID string `json:"sub_id"`
	}

	// SubscribedResult is sent to clients with current state on subscribe
	SubscribedResult struct {
		Type           string           `json:"type"`
		SubscriptionID string           `json:"sub_id,omitempty"`
		Items          []SubscribedItem `json:"items"`
	}

	// SubscribedItem is one projected aggregate state on subscribe
	SubscribedItem struct {
		AggregateID []string        `json:"id"`
		Data        json.RawMessage `json:"data"`
		Sequence    int64           `json:"sequence"`
	}
)
