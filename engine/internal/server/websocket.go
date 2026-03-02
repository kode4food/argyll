package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Client represents a WebSocket client connection for event streaming
	Client struct {
		hub           *timebox.EventHub
		conn          *websocket.Conn
		consumer      *timebox.Consumer
		getState      StateFunc
		subscriptions map[string]*clientSubscription
		onClose       func(*Client)
		closeOnce     sync.Once
	}
	// StateFunc retrieves the current projected state and next sequence for an
	// aggregate. The next sequence is used by clients to detect sequence skew
	StateFunc func(timebox.AggregateID) (any, int64, error)

	// RegisterFunc registers a client with the caller
	RegisterFunc func(*Client)

	clientMessage struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	clientSubscription struct {
		id          string
		aggregateID timebox.AggregateID
		eventTypes  []timebox.EventType
		minSeq      int64
	}
)

const (
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10
	maxMessageSize     = 512
	wsBufferSize       = 1024
	incomingBufferSize = 16
)

var (
	ErrMissingSubscriptionID = errors.New("missing sub_id")
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  wsBufferSize,
		WriteBufferSize: wsBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// HandleWebSocket upgrades an HTTP connection to WebSocket and starts
// streaming events based on client subscriptions
func HandleWebSocket(
	hub *timebox.EventHub, w http.ResponseWriter, r *http.Request,
	st StateFunc, register RegisterFunc,
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", log.Error(err))
		return
	}

	client := &Client{
		hub:           hub,
		conn:          conn,
		consumer:      hub.NewConsumer(),
		getState:      st,
		subscriptions: map[string]*clientSubscription{},
	}

	if register != nil {
		register(client)
	}

	go client.run()
}

func (s *Server) handleWebSocket(c *gin.Context) {
	HandleWebSocket(s.eventHub, c.Writer, c.Request,
		s.lookupSubscriptionState,
		func(c *Client) {
			c.onClose = s.unregisterWebSocket
			s.registerWebSocket(c)
		},
	)
}

func (s *Server) lookupSubscriptionState(
	id timebox.AggregateID,
) (any, int64, error) {
	if len(id) == 0 {
		return nil, 0, nil
	}
	switch string(id[0]) {
	case events.CatalogPrefix:
		if len(id) == 1 {
			return s.engine.GetCatalogStateSeq()
		}
	case events.PartitionPrefix:
		if len(id) == 1 {
			return s.engine.GetPartitionStateSeq()
		}
	case events.FlowPrefix:
		if len(id) == 2 {
			flowID := api.FlowID(id[1])
			return s.engine.GetFlowStateSeq(flowID)
		}
	}
	return nil, 0, nil
}

func (c *Client) run() {
	defer func() {
		c.Close()
		if c.onClose != nil {
			c.onClose(c)
		}
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
			if !c.handleMessage(message) {
				return
			}

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

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.consumer != nil {
			c.consumer.Close()
		}
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
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

func (c *Client) handleMessage(message []byte) bool {
	var env clientMessage
	if err := json.Unmarshal(message, &env); err != nil {
		slog.Error("Failed to parse WebSocket message", log.Error(err))
		return true
	}

	switch env.Type {
	case "subscribe":
		var sub api.ClientSubscription
		if err := json.Unmarshal(env.Data, &sub); err != nil {
			slog.Error("Failed to parse subscribe payload", log.Error(err))
			return true
		}
		c.addSubscription(&sub)
	case "unsubscribe":
		var sub api.ClientUnsubscription
		if err := json.Unmarshal(env.Data, &sub); err != nil {
			slog.Error("Failed to parse unsubscribe payload", log.Error(err))
			return true
		}
		c.removeSubscription(sub.SubscriptionID)
	default:
		return true
	}
	return true
}

func (c *Client) addSubscription(sub *api.ClientSubscription) {
	cs, err := newClientSubscription(sub)
	if err != nil {
		slog.Error("Failed to register subscription", log.Error(err))
		return
	}
	c.subscriptions[cs.id] = cs

	if len(cs.aggregateID) > 0 {
		c.sendSubscribeState(cs)
	}
}

func (c *Client) removeSubscription(subscriptionID string) {
	delete(c.subscriptions, subscriptionID)
}

func (c *Client) sendSubscribeState(sub *clientSubscription) {
	if c.getState == nil {
		return
	}

	state, nextSeq, err := c.getState(sub.aggregateID)
	if err != nil {
		slog.Error("Failed to get state for subscription",
			slog.Any("aggregate_id", sub.aggregateID),
			log.Error(err))
		return
	}

	data, err := json.Marshal(state)
	if err != nil {
		slog.Error("Failed to marshal state",
			slog.Any("aggregate_id", sub.aggregateID),
			log.Error(err))
		return
	}

	sub.minSeq = nextSeq

	msg := api.SubscribedResult{
		Type:           "subscribed",
		AggregateID:    idToStrings(sub.aggregateID),
		SubscriptionID: sub.id,
		Data:           data,
		Sequence:       nextSeq,
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(msg); err != nil {
		slog.Error("WebSocket write failed",
			slog.String("context", "subscribed"),
			log.Error(err))
	}
}

func (c *Client) sendEventIfMatched(event *timebox.Event) bool {
	if event == nil {
		return true
	}

	for _, sub := range c.subscriptions {
		if !sub.matches(event) || event.Sequence < sub.minSeq {
			continue
		}

		wsEvent := c.transformEvent(event, sub.id)
		_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteJSON(wsEvent); err != nil {
			slog.Error("WebSocket write failed", log.Error(err))
			return false
		}
	}
	return true
}

func (c *Client) transformEvent(
	ev *timebox.Event, subscriptionID string,
) *api.WebSocketEvent {
	return &api.WebSocketEvent{
		Type:           api.EventType(ev.Type),
		Data:           ev.Data,
		Timestamp:      ev.Timestamp.UnixMilli(),
		AggregateID:    idToStrings(ev.AggregateID),
		SubscriptionID: subscriptionID,
		Sequence:       ev.Sequence,
	}
}

func (c *Client) sendPing() bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

func subscriptionEventTypes(
	sub *api.ClientSubscription,
) []timebox.EventType {
	if len(sub.EventTypes) == 0 {
		return nil
	}
	eventTypes := make([]timebox.EventType, len(sub.EventTypes))
	for i, eventType := range sub.EventTypes {
		eventTypes[i] = timebox.EventType(eventType)
	}
	return eventTypes
}

func newClientSubscription(
	sub *api.ClientSubscription,
) (*clientSubscription, error) {
	if sub.SubscriptionID == "" {
		return nil, ErrMissingSubscriptionID
	}
	return &clientSubscription{
		id:          sub.SubscriptionID,
		aggregateID: stringsToID(sub.AggregateID),
		eventTypes:  subscriptionEventTypes(sub),
	}, nil
}

func (s *clientSubscription) matches(ev *timebox.Event) bool {
	if len(s.aggregateID) > 0 && !ev.AggregateID.HasPrefix(s.aggregateID) {
		return false
	}
	if len(s.eventTypes) == 0 {
		return true
	}
	return slices.Contains(s.eventTypes, ev.Type)
}

// BuildFilter creates an event filter based on client subscription preferences
// for event types and aggregate IDs
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
