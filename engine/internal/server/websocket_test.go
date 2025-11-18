package server

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
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/pkg/api"
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

func testWebSocket(t *testing.T, replay ReplayFunc) *testWebSocketEnv {
	t.Helper()
	hub := &mockEventHub{}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			HandleWebSocket(hub, w, r, replay)
		},
	))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	return &testWebSocketEnv{
		Server: server,
		Hub:    hub,
		Conn:   conn,
	}
}

func eventTypePtr(et timebox.EventType) *timebox.EventType {
	return &et
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
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		Data:        json.RawMessage(`{"test":"data"}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	env.Hub.Send(event)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	require.NoError(t, err)

	assert.Equal(t, api.EventTypeFlowStarted, wsEvent.Type)
	assert.Equal(t, json.RawMessage(`{"test":"data"}`), wsEvent.Data)
}

func TestMessageInvalid(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	err := env.Conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
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
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	event := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
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
			Type:        api.EventTypeFlowStarted,
			Data:        json.RawMessage(`{"replayed":true}`),
			Timestamp:   time.Now(),
			AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		},
		{
			Type:        api.EventTypeStepCompleted,
			Data:        json.RawMessage(`{"step":"test"}`),
			Timestamp:   time.Now(),
			AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		},
	}

	replay := func(flowID timebox.ID, fromSeq int64) ([]*timebox.Event, error) {
		assert.Equal(t, timebox.ID("wf-123"), flowID)
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
	require.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var wsEvent1 api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent1)
	require.NoError(t, err)
	assert.Equal(t, api.EventTypeFlowStarted, wsEvent1.Type)

	var wsEvent2 api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent2)
	require.NoError(t, err)
	assert.Equal(t, api.EventTypeStepCompleted, wsEvent2.Type)
}

func TestReplayWithError(t *testing.T) {
	replay := func(flowID timebox.ID, fromSeq int64) ([]*timebox.Event, error) {
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
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	if err == nil {
		t.Fatal("Should not receive events when replay fails")
	}
}

func TestReplayWithoutFlowID(t *testing.T) {
	replayCalled := false
	replay := func(flowID timebox.ID, fromSeq int64) ([]*timebox.Event, error) {
		replayCalled = true
		return nil, nil
	}

	env := testWebSocket(t, replay)
	defer env.Cleanup()

	sub := api.SubscribeMessage{
		Type: "subscribe",
		Data: api.ClientSubscription{
			EventTypes: []*timebox.EventType{eventTypePtr(api.EventTypeFlowStarted)},
		},
	}
	err := env.Conn.WriteJSON(sub)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.False(t, replayCalled)
}

func TestEngineEvents(t *testing.T) {
	sub := &api.ClientSubscription{
		EngineEvents: true,
	}

	filter := BuildFilter(sub)

	engineEvent := &timebox.Event{
		Type:        api.EventTypeStepRegistered,
		AggregateID: timebox.NewAggregateID("engine", "engine"),
	}
	flowEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(engineEvent))
	assert.False(t, filter(flowEvent))
}

func TestEventTypes(t *testing.T) {
	sub := &api.ClientSubscription{
		EventTypes: []*timebox.EventType{
			eventTypePtr(api.EventTypeFlowStarted),
			eventTypePtr(api.EventTypeStepCompleted),
		},
	}

	filter := BuildFilter(sub)

	createdEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	executedEvent := &timebox.Event{
		Type:        api.EventTypeStepCompleted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        api.EventTypeFlowCompleted,
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

	filter := BuildFilter(sub)

	matchingEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-456"),
	}

	assert.True(t, filter(matchingEvent))
	assert.False(t, filter(otherEvent))
}

func TestNoFilters(t *testing.T) {
	sub := &api.ClientSubscription{}

	filter := BuildFilter(sub)

	event := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.False(t, filter(event))
}

func TestCombined(t *testing.T) {
	sub := &api.ClientSubscription{
		EngineEvents: true,
		EventTypes:   []*timebox.EventType{eventTypePtr(api.EventTypeFlowStarted)},
	}

	filter := BuildFilter(sub)

	engineEvent := &timebox.Event{
		Type:        api.EventTypeStepRegistered,
		AggregateID: timebox.NewAggregateID("engine", "engine"),
	}
	flowEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	otherEvent := &timebox.Event{
		Type:        api.EventTypeStepCompleted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(engineEvent))
	assert.True(t, filter(flowEvent))
	assert.False(t, filter(otherEvent))
}

func TestEventTypesWithFlowID(t *testing.T) {
	sub := &api.ClientSubscription{
		FlowID:     "wf-123",
		EventTypes: []*timebox.EventType{eventTypePtr(api.EventTypeFlowStarted)},
	}

	filter := BuildFilter(sub)

	matchingEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongTypeEvent := &timebox.Event{
		Type:        api.EventTypeStepCompleted,
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongFlowEvent := &timebox.Event{
		Type:        api.EventTypeFlowStarted,
		AggregateID: timebox.NewAggregateID("flow", "wf-456"),
	}

	assert.True(t, filter(matchingEvent))
	assert.False(t, filter(wrongTypeEvent))
	assert.True(t, filter(wrongFlowEvent))
}
