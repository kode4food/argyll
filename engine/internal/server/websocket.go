package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	Client struct {
		hub      timebox.EventHub
		conn     *websocket.Conn
		consumer topic.Consumer[*timebox.Event]
		filter   events.EventFilter
		replay   ReplayFunc
	}

	ReplayFunc func(flowID timebox.ID, fromSeq int64) ([]*timebox.Event, error)
)

const (
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10
	maxMessageSize     = 512
	wsBufferSize       = 1024
	incomingBufferSize = 16
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  wsBufferSize,
	WriteBufferSize: wsBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *Client) transformEvent(ev *timebox.Event) *api.WebSocketEvent {
	return &api.WebSocketEvent{
		Type:        ev.Type,
		Data:        ev.Data,
		Timestamp:   ev.Timestamp.UnixMilli(),
		AggregateID: ev.AggregateID,
		Sequence:    ev.Sequence,
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and starts
// streaming events based on client subscriptions
func HandleWebSocket(
	hub timebox.EventHub, w http.ResponseWriter, r *http.Request,
	replay ReplayFunc,
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed",
			slog.Any("error", err))
		return
	}

	noopFilter := func(*timebox.Event) bool { return false }
	client := &Client{
		hub:      hub,
		conn:     conn,
		consumer: hub.NewConsumer(),
		filter:   noopFilter,
		replay:   replay,
	}

	go client.run()
}

func (c *Client) run() {
	defer func() {
		c.consumer.Close()
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	incoming := make(chan []byte, incomingBufferSize)
	go c.readMessages(incoming)

	for {
		select {
		case message, ok := <-incoming:
			if !ok {
				return
			}
			c.handleSubscribe(message)

		case event, ok := <-c.consumer.Receive():
			if !ok {
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if !c.sendEventIfMatched(event) {
				return
			}

		case <-ticker.C:
			if !c.sendPing() {
				return
			}
		}
	}
}

func (c *Client) readMessages(incoming chan []byte) {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			close(incoming)
			return
		}
		incoming <- message
	}
}

func (c *Client) handleSubscribe(message []byte) {
	var sub api.SubscribeMessage
	if err := json.Unmarshal(message, &sub); err != nil {
		slog.Error("Failed to parse WebSocket message",
			slog.Any("error", err))
		return
	}

	if sub.Type != "subscribe" {
		return
	}

	c.filter = BuildFilter(&sub.Data)

	if sub.Data.FlowID != "" && sub.Data.FromSequence >= 0 {
		c.replayAndSend(sub.Data.FlowID, sub.Data.FromSequence)
	}
}

func (c *Client) sendEventIfMatched(event *timebox.Event) bool {
	if !c.filter(event) {
		return true
	}

	wsEvent := c.transformEvent(event)
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(wsEvent); err != nil {
		slog.Error("WebSocket write failed",
			slog.Any("error", err))
		return false
	}
	return true
}

func (c *Client) sendPing() bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

func (c *Client) replayAndSend(flowID timebox.ID, fromSeq int64) {
	if c.replay == nil {
		return
	}

	evs, err := c.replay(flowID, fromSeq)
	if err != nil {
		slog.Error("Failed to replay workflow events",
			slog.Any("flow_id", flowID),
			slog.Int64("from_sequence", fromSeq),
			slog.Any("error", err))
		return
	}

	for _, ev := range evs {
		if !c.writeEvent(ev, "replay") {
			return
		}
	}
}

func (c *Client) writeEvent(ev *timebox.Event, context string) bool {
	wsEvent := c.transformEvent(ev)
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteJSON(wsEvent)
	if err != nil {
		slog.Error("WebSocket write failed",
			slog.String("context", context),
			slog.Any("error", err))
		return false
	}
	return true
}

// BuildFilter creates an event filter based on client subscription preferences
// for event types, workflow IDs, or engine events
func BuildFilter(sub *api.ClientSubscription) events.EventFilter {
	var filters []events.EventFilter

	if sub.EngineEvents {
		filters = append(filters, events.IsEngineEvent)
	}

	if len(sub.EventTypes) > 0 {
		eventTypes := make([]timebox.EventType, len(sub.EventTypes))
		for i, et := range sub.EventTypes {
			eventTypes[i] = *et
		}
		filters = append(filters, events.FilterEvents(eventTypes...))
	}

	if len(sub.EventTypes) == 0 && sub.FlowID != "" {
		filters = append(filters, events.FilterWorkflow(sub.FlowID))
	}

	if len(filters) == 0 {
		return func(*timebox.Event) bool { return false }
	}

	return events.OrFilters(filters...)
}
