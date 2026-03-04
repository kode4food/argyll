package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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
		getState      StateFunc
		subscriptions map[string]*clientSubscription
		subMu         sync.Mutex
		writeMu       sync.Mutex
		onClose       func(*Client)
		closeOnce     sync.Once
		done          chan struct{}
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
		id           string
		aggregateIDs []timebox.AggregateID
		includeState bool
		eventTypes   []timebox.EventType
		minSeqs      map[string]int64
		consumer     *timebox.Consumer
		active       atomic.Bool
	}
)

const (
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10
	maxMessageSize     = 64 * 1024
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
		getState:      st,
		subscriptions: map[string]*clientSubscription{},
		done:          make(chan struct{}),
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

		case <-c.done:
			return

		case <-ticker.C:
			if !c.sendPing() {
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)

		c.subMu.Lock()
		for _, sub := range c.subscriptions {
			sub.close()
		}
		c.subscriptions = map[string]*clientSubscription{}
		c.subMu.Unlock()

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
	cs, err := newClientSubscription(c.hub, sub)
	if err != nil {
		slog.Error("Failed to register subscription", log.Error(err))
		return
	}

	c.subMu.Lock()
	if prev := c.subscriptions[cs.id]; prev != nil {
		prev.close()
	}
	c.subscriptions[cs.id] = cs
	c.subMu.Unlock()

	if !c.sendSubscribeState(cs) {
		cs.close()
		c.subMu.Lock()
		delete(c.subscriptions, cs.id)
		c.subMu.Unlock()
		c.Close()
		return
	}

	go c.streamSubscription(cs)
}

func (c *Client) removeSubscription(subscriptionID string) {
	c.subMu.Lock()
	sub := c.subscriptions[subscriptionID]
	delete(c.subscriptions, subscriptionID)
	c.subMu.Unlock()

	if sub != nil {
		sub.close()
	}
}

func (c *Client) streamSubscription(sub *clientSubscription) {
	for event := range sub.consumer.Receive() {
		if !sub.active.Load() {
			return
		}

		if !c.writeSubscriptionEvent(sub, event) {
			c.Close()
			return
		}
	}
}

func (c *Client) sendSubscribeState(sub *clientSubscription) bool {
	if !sub.includeState || c.getState == nil {
		return true
	}

	items := make([]api.SubscribedItem, 0, len(sub.aggregateIDs))
	sub.minSeqs = make(map[string]int64, len(sub.aggregateIDs))
	for _, id := range sub.aggregateIDs {
		state, nextSeq, err := c.getState(id)
		if err != nil {
			slog.Error("Failed to get state for subscription",
				slog.Any("aggregate_id", id),
				log.Error(err))
			return false
		}

		data, err := json.Marshal(state)
		if err != nil {
			slog.Error("Failed to marshal state",
				slog.Any("aggregate_id", id),
				log.Error(err))
			return false
		}

		items = append(items, api.SubscribedItem{
			AggregateID: idToStrings(id),
			Data:        data,
			Sequence:    nextSeq,
		})
		sub.minSeqs[aggregateIDKey(id)] = nextSeq
	}

	msg := api.SubscribedResult{
		Type:           "subscribed",
		SubscriptionID: sub.id,
		Items:          items,
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteJSON(msg)
	if err != nil {
		slog.Error("WebSocket write failed",
			slog.String("context", "subscribed"),
			log.Error(err))
		return false
	}
	return true
}

func (c *Client) writeSubscriptionEvent(
	sub *clientSubscription, event *timebox.Event,
) bool {
	if !sub.active.Load() {
		return true
	}

	if minSeq, ok := sub.minSeqs[aggregateIDKey(event.AggregateID)]; ok &&
		event.Sequence < minSeq {
		return true
	}

	wsEvent := c.transformEvent(event)
	wsEvent.SubscriptionID = sub.id

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteJSON(wsEvent)
	if err != nil {
		slog.Error("WebSocket write failed", log.Error(err))
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
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

func subscriptionEventTypes(sub *api.ClientSubscription) []timebox.EventType {
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
	hub *timebox.EventHub, sub *api.ClientSubscription,
) (*clientSubscription, error) {
	if sub.SubscriptionID == "" {
		return nil, ErrMissingSubscriptionID
	}
	res := &clientSubscription{
		id:           sub.SubscriptionID,
		aggregateIDs: stringsToIDs(sub.AggregateIDs),
		includeState: sub.IncludeState,
		eventTypes:   subscriptionEventTypes(sub),
	}
	res.consumer = hub.NewAggregatesConsumer(res.aggregateIDs, res.eventTypes...)
	res.active.Store(true)
	return res, nil
}

func (s *clientSubscription) close() {
	s.active.Store(false)
	s.consumer.Close()
}

func idToStrings(id timebox.AggregateID) []string {
	res := make([]string, len(id))
	for i, p := range id {
		res[i] = string(p)
	}
	return res
}

func stringsToIDs(parts [][]string) []timebox.AggregateID {
	if len(parts) == 0 {
		return nil
	}
	res := make([]timebox.AggregateID, 0, len(parts))
	for _, p := range parts {
		res = append(res, stringsToID(p))
	}
	return res
}

func aggregateIDKey(id timebox.AggregateID) string {
	return strings.Join(idToStrings(id), "\x00")
}

func stringsToID(parts []string) timebox.AggregateID {
	res := make(timebox.AggregateID, 0, len(parts))
	for _, part := range parts {
		res = append(res, timebox.ID(part))
	}
	return res
}
