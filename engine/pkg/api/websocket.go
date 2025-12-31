package api

import "encoding/json"

type (
	// WebSocketEvent is an event sent to WebSocket clients
	WebSocketEvent struct {
		Type        EventType       `json:"type"`
		Data        json.RawMessage `json:"data"`
		AggregateID []string        `json:"id"`
		Timestamp   int64           `json:"timestamp"`
		Sequence    int64           `json:"sequence"`
	}

	// SubscribeMessage is sent by clients to subscribe to events
	SubscribeMessage struct {
		Type string             `json:"type"`
		Data ClientSubscription `json:"data"`
	}

	// ClientSubscription configures which events a WebSocket client receives
	ClientSubscription struct {
		AggregateID []string    `json:"aggregate_id"`
		EventTypes  []EventType `json:"event_types,omitempty"`
	}

	// SubscribeState is sent to clients with current state on subscribe
	SubscribeState struct {
		Type        string          `json:"type"`
		AggregateID []string        `json:"id"`
		Data        json.RawMessage `json:"data"`
		Sequence    int64           `json:"sequence"`
	}
)
