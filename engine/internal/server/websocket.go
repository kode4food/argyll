package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Client represents a WebSocket client connection for event streaming
	Client struct {
		hub      timebox.EventHub
		conn     *websocket.Conn
		consumer topic.Consumer[*timebox.Event]
		filter   events.EventFilter
		getState StateFunc
		minSeq   int64
	}

	// StateFunc retrieves the current projected state and next sequence for an
	// aggregate. The next sequence is used by clients to detect sequence skew
	StateFunc func(context.Context, timebox.AggregateID) (any, int64, error)
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

// HandleWebSocket upgrades an HTTP connection to WebSocket and starts
// streaming events based on client subscriptions
func HandleWebSocket(
	hub timebox.EventHub, w http.ResponseWriter, r *http.Request, st StateFunc,
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed",
			log.Error(err))
		return
	}

	noopFilter := func(*timebox.Event) bool { return false }
	client := &Client{
		hub:      hub,
		conn:     conn,
		consumer: hub.NewConsumer(),
		filter:   noopFilter,
		getState: st,
	}

	go client.run()
}

func (s *Server) handleWebSocket(c *gin.Context) {
	HandleWebSocket(s.eventHub, c.Writer, c.Request,
		func(ctx context.Context, id timebox.AggregateID) (any, int64, error) {
			if len(id) == 0 {
				return nil, 0, nil
			}
			switch string(id[0]) {
			case "engine":
				return s.engine.GetEngineStateSeq(ctx)
			case "flow":
				if len(id) < 2 {
					return nil, 0, errors.New("invalid aggregate_id")
				}
				flowID := api.FlowID(id[1])
				return s.engine.GetFlowStateSeq(ctx, flowID)
			default:
				return nil, 0, errors.New("invalid aggregate_id")
			}
		},
	)
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
	var sub api.SubscribeRequest
	if err := json.Unmarshal(message, &sub); err != nil {
		slog.Error("Failed to parse WebSocket message",
			log.Error(err))
		return
	}

	if sub.Type != "subscribe" {
		return
	}

	c.filter = BuildFilter(&sub.Data)

	if len(sub.Data.AggregateID) > 0 {
		c.sendSubscribeState(stringsToID(sub.Data.AggregateID))
	}
}

func (c *Client) sendSubscribeState(aggregateID timebox.AggregateID) {
	if c.getState == nil {
		return
	}

	state, nextSeq, err := c.getState(context.Background(), aggregateID)
	if err != nil {
		slog.Error("Failed to get state for subscription",
			slog.Any("aggregate_id", aggregateID),
			log.Error(err))
		return
	}

	data, err := json.Marshal(state)
	if err != nil {
		slog.Error("Failed to marshal state",
			slog.Any("aggregate_id", aggregateID),
			log.Error(err))
		return
	}

	c.minSeq = nextSeq

	msg := api.SubscribedResult{
		Type:        "subscribed",
		AggregateID: idToStrings(aggregateID),
		Data:        data,
		Sequence:    nextSeq,
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(msg); err != nil {
		slog.Error("WebSocket write failed",
			slog.String("context", "subscribed"),
			log.Error(err))
	}
}

func (c *Client) sendEventIfMatched(event *timebox.Event) bool {
	if event.Sequence < c.minSeq || !c.filter(event) {
		return true
	}

	wsEvent := c.transformEvent(event)
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(wsEvent); err != nil {
		slog.Error("WebSocket write failed",
			log.Error(err))
		return false
	}
	return true
}

func (c *Client) transformEvent(ev *timebox.Event) *api.WebSocketEvent {
	return &api.WebSocketEvent{
		Type:        api.EventType(ev.Type),
		Data:        ev.Data,
		Timestamp:   ev.Timestamp.UnixMilli(),
		AggregateID: idToStrings(ev.AggregateID),
		Sequence:    ev.Sequence,
	}
}

func (c *Client) sendPing() bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

// BuildFilter creates an event filter based on client subscription preferences
// for event types and aggregate IDs
func BuildFilter(sub *api.ClientSubscription) events.EventFilter {
	var aggregateFilter events.EventFilter
	if len(sub.AggregateID) > 0 {
		id := stringsToID(sub.AggregateID)
		aggregateFilter = events.FilterAggregate(id)
	}

	var eventTypeFilter events.EventFilter
	if len(sub.EventTypes) > 0 {
		timeboxEventTypes := make([]timebox.EventType, len(sub.EventTypes))
		for i, et := range sub.EventTypes {
			timeboxEventTypes[i] = timebox.EventType(et)
		}
		eventTypeFilter = events.FilterEvents(timeboxEventTypes...)
	}

	switch {
	case aggregateFilter != nil && eventTypeFilter != nil:
		return events.AndFilters(aggregateFilter, eventTypeFilter)
	case aggregateFilter != nil:
		return aggregateFilter
	case eventTypeFilter != nil:
		return eventTypeFilter
	default:
		return func(*timebox.Event) bool { return false }
	}
}

func idToStrings(id timebox.AggregateID) []string {
	res := make([]string, len(id))
	for i, p := range id {
		res[i] = string(p)
	}
	return res
}

func stringsToID(parts []string) timebox.AggregateID {
	res := make(timebox.AggregateID, 0, len(parts))
	for _, part := range parts {
		res = append(res, timebox.ID(part))
	}
	return res
}
