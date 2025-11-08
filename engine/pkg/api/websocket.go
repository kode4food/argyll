package api

import (
	"encoding/json"

	"github.com/kode4food/timebox"
)

type (
	// WebSocketEvent is an event sent to WebSocket clients
	WebSocketEvent struct {
		Type        timebox.EventType   `json:"type"`
		Data        json.RawMessage     `json:"data"`
		AggregateID timebox.AggregateID `json:"aggregate_id"`
		Timestamp   int64               `json:"timestamp"`
		Sequence    int64               `json:"sequence"`
	}

	// ClientSubscription configures which events a WebSocket client receives
	ClientSubscription struct {
		WorkflowID   timebox.ID           `json:"workflow_id"`
		EventTypes   []*timebox.EventType `json:"event_types,omitempty"`
		FromSequence int64                `json:"from_sequence,omitempty"`
		EngineEvents bool                 `json:"engine_events,omitempty"`
	}

	// SubscribeMessage is sent by clients to subscribe to events
	SubscribeMessage struct {
		Type string             `json:"type"`
		Data ClientSubscription `json:"data"`
	}
)
