package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	mockEventHub struct {
		consumers []*mockConsumer
	}

	mockConsumer struct {
		ch       chan *timebox.Event
		closedCh chan struct{}
	}

	testWebSocketEnv struct {
		Server *httptest.Server
		Hub    *mockEventHub
		Conn   *websocket.Conn
	}
)

func (m *mockEventHub) Length() uint64 {
	return 0
}

func (m *mockEventHub) NewProducer() topic.Producer[*timebox.Event] {
	return nil
}

func (m *mockEventHub) NewConsumer() topic.Consumer[*timebox.Event] {
	consumer := &mockConsumer{
		ch:       make(chan *timebox.Event, 10),
		closedCh: make(chan struct{}),
	}
	m.consumers = append(m.consumers, consumer)
	return consumer
}

func (m *mockEventHub) Send(event *timebox.Event) {
	for _, c := range m.consumers {
		select {
		case <-c.closedCh:
		case c.ch <- event:
		default:
		}
	}
}

func (m *mockConsumer) Receive() <-chan *timebox.Event {
	return m.ch
}

func (m *mockConsumer) IsClosed() <-chan struct{} {
	return m.closedCh
}

func (m *mockConsumer) Close() {
	select {
	case <-m.closedCh:
	default:
		close(m.closedCh)
		close(m.ch)
	}
}

func (e *testWebSocketEnv) Cleanup() {
	if e.Conn != nil {
		_ = e.Conn.Close()
	}
	if e.Server != nil {
		e.Server.Close()
	}
}

func TestHandleWebSocket(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err := env.Conn.ReadMessage()
	if err == nil {
		t.Fatal("Expected timeout reading message")
	}
}

func TestClientReceivesEvent(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "subscribe",
		Data: api.ClientSubscription{
			FlowID: "wf-123",
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        json.RawMessage(`{"test":"data"}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	env.Hub.Send(event)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)

	assert.Equal(t, api.EventTypeFlowStarted, wsEvent.Type)
	assert.Equal(t, json.RawMessage(`{"test":"data"}`), wsEvent.Data)
}

func TestMessageInvalid(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	err := env.Conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        json.RawMessage(`{}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	env.Hub.Send(event)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	if err == nil {
		t.Fatal("Should not receive event with invalid subscription")
	}
}

func TestMessageNonSubscribe(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "other",
		Data: api.ClientSubscription{
			FlowID: "wf-123",
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        json.RawMessage(`{}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	env.Hub.Send(event)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	if err == nil {
		t.Fatal("Should not receive event without valid subscription")
	}
}

func TestReplayWithEvents(t *testing.T) {
	replayEvents := []*timebox.Event{
		{
			Type:        timebox.EventType(api.EventTypeFlowStarted),
			Data:        json.RawMessage(`{"replayed":true}`),
			Timestamp:   time.Now(),
			AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		},
		{
			Type:        timebox.EventType(api.EventTypeStepCompleted),
			Data:        json.RawMessage(`{"step":"test"}`),
			Timestamp:   time.Now(),
			AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		},
	}

	replay := func(flowID api.FlowID, fromSeq int64) ([]*timebox.Event, error) {
		assert.Equal(t, api.FlowID("wf-123"), flowID)
		assert.Equal(t, int64(0), fromSeq)
		return replayEvents, nil
	}

	env := testWebSocket(t, replay)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "subscribe",
		Data: api.ClientSubscription{
			FlowID: "wf-123",
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var wsEvent1 api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent1)
	assert.NoError(t, err)
	assert.Equal(t, api.EventTypeFlowStarted, wsEvent1.Type)

	var wsEvent2 api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent2)
	assert.NoError(t, err)
	assert.Equal(t, api.EventTypeStepCompleted, wsEvent2.Type)
}

func TestReplayWithError(t *testing.T) {
	replay := func(flowID api.FlowID, fromSeq int64) ([]*timebox.Event, error) {
		return nil, assert.AnError
	}

	env := testWebSocket(t, replay)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "subscribe",
		Data: api.ClientSubscription{
			FlowID: "wf-123",
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	if err == nil {
		t.Fatal("Should not receive events when replay fails")
	}
}

func TestReplayWithoutFlowID(t *testing.T) {
	replayCalled := false
	replay := func(flowID api.FlowID, fromSeq int64) ([]*timebox.Event, error) {
		replayCalled = true
		return nil, nil
	}

	env := testWebSocket(t, replay)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "subscribe",
		Data: api.ClientSubscription{
			EventTypes: []api.EventType{
				api.EventTypeFlowStarted,
			},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.False(t, replayCalled)
}

func TestEngineEvents(t *testing.T) {
	sub := &api.ClientSubscription{
		EngineEvents: true,
	}

	filter := server.BuildFilter(sub)

	engineEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("engine", "engine"),
	}
	flowEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(engineEvent))
	assert.False(t, filter(flowEvent))
}

func TestEventTypes(t *testing.T) {
	sub := &api.ClientSubscription{
		EventTypes: []api.EventType{
			api.EventTypeFlowStarted,
			api.EventTypeStepCompleted,
		},
	}

	filter := server.BuildFilter(sub)

	createdEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	executedEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepCompleted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowCompleted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(createdEvent))
	assert.True(t, filter(executedEvent))
	assert.False(t, filter(otherEvent))
}

func TestFlow(t *testing.T) {
	sub := &api.ClientSubscription{
		FlowID: "wf-123",
	}

	filter := server.BuildFilter(sub)

	matchingEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-456"),
	}

	assert.True(t, filter(matchingEvent))
	assert.False(t, filter(otherEvent))
}

func TestNoFilters(t *testing.T) {
	sub := &api.ClientSubscription{}

	filter := server.BuildFilter(sub)

	event := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.False(t, filter(event))
}

func TestCombined(t *testing.T) {
	sub := &api.ClientSubscription{
		EngineEvents: true,
		EventTypes:   []api.EventType{api.EventTypeFlowStarted},
	}

	filter := server.BuildFilter(sub)

	engineEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("engine", "engine"),
	}
	flowEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepCompleted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(engineEvent))
	assert.True(t, filter(flowEvent))
	assert.False(t, filter(otherEvent))
}

func TestEventTypesWithFlowID(t *testing.T) {
	sub := &api.ClientSubscription{
		FlowID:     "wf-123",
		EventTypes: []api.EventType{api.EventTypeFlowStarted},
	}

	filter := server.BuildFilter(sub)

	matchingEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongTypeEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepCompleted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongFlowEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-456"),
	}

	assert.True(t, filter(matchingEvent))
	assert.False(t, filter(wrongTypeEvent))
	assert.True(t, filter(wrongFlowEvent))
}

func testWebSocket(t *testing.T, replay server.ReplayFunc) *testWebSocketEnv {
	t.Helper()
	hub := &mockEventHub{}

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			server.HandleWebSocket(hub, w, r, replay)
		},
	))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	return &testWebSocketEnv{
		Server: srv,
		Hub:    hub,
		Conn:   conn,
	}
}
