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

	serverWebSocketEnv struct {
		Server *httptest.Server
		Conn   *websocket.Conn
	}
)

const (
	wsReadTimeout  = 500 * time.Millisecond
	wsCloseTimeout = 200 * time.Millisecond
	wsStateTimeout = 500 * time.Millisecond
	wsErrorTimeout = 100 * time.Millisecond
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

func (e *serverWebSocketEnv) Cleanup() {
	if e.Conn != nil {
		_ = e.Conn.Close()
	}
	if e.Server != nil {
		e.Server.Close()
	}
}

func TestSocket(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err := env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestClientReceivesEvent(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)

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
	assert.Error(t, err)
}

func TestMessageNonSubscribe(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "other",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

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
	assert.Error(t, err)
}

func TestSubscribeStateSendsState(t *testing.T) {
	flowState := &api.FlowState{
		ID:     "wf-123",
		Status: api.FlowActive,
	}

	getState := func(id timebox.AggregateID) (any, int64, error) {
		assert.Equal(t, timebox.NewAggregateID("flow", "wf-123"), id)
		return flowState, 5, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	assert.Equal(t, "subscribed", stateMsg.Type)
	assert.Equal(t, []string{"flow", "wf-123"}, stateMsg.AggregateID)
	assert.Equal(t, int64(5), stateMsg.Sequence)

	var receivedState api.FlowState
	err = json.Unmarshal(stateMsg.Data, &receivedState)
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("wf-123"), receivedState.ID)
	assert.Equal(t, api.FlowActive, receivedState.Status)
}

func TestStaleEventsFiltered(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 10, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), stateMsg.Sequence)

	// Send stale event (sequence 5 < minSequence 10)
	staleEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		Data:        json.RawMessage(`{"stale":true}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		Sequence:    5,
	}
	env.Hub.Send(staleEvent)

	// Send fresh event (sequence 10 >= minSequence 10)
	freshEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepStarted),
		Data:        json.RawMessage(`{"fresh":true}`),
		Timestamp:   time.Now(),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
		Sequence:    10,
	}
	env.Hub.Send(freshEvent)

	// Should only receive the fresh event
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, api.EventTypeStepStarted, wsEvent.Type)
	assert.Equal(t, json.RawMessage(`{"fresh":true}`), wsEvent.Data)
}

func TestSubscribeStateWithError(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		return nil, 0, assert.AnError
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestSubscribeNoID(t *testing.T) {
	getStateCalled := false
	getState := func(id timebox.AggregateID) (any, int64, error) {
		getStateCalled = true
		return nil, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			EventTypes: []api.EventType{
				api.EventTypeFlowStarted,
			},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	assert.False(t, getStateCalled)
}

func TestClientPongHandler(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	err := env.Conn.WriteMessage(websocket.PongMessage, []byte("pong"))
	assert.NoError(t, err)

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err = env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	assert.Equal(t, []string{"flow", "wf-123"}, stateMsg.AggregateID)
}

func TestClientConsumerClosed(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	assert.Len(t, env.Hub.consumers, 1)
	env.Hub.consumers[0].Close()

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsCloseTimeout))
	_, _, err := env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestSocketCallbackEngine(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		sub := api.SubscribeRequest{
			Type: "subscribe",
			Data: api.ClientSubscription{
				AggregateID: []string{"engine"},
			},
		}
		err := ws.Conn.WriteJSON(sub)
		assert.NoError(t, err)

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
		var stateMsg api.SubscribedResult
		err = ws.Conn.ReadJSON(&stateMsg)
		assert.NoError(t, err)
		assert.Equal(t, []string{"engine"}, stateMsg.AggregateID)

		var engState api.EngineState
		err = json.Unmarshal(stateMsg.Data, &engState)
		assert.NoError(t, err)
	})
}

func TestSocketCallbackFlow(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		err := env.Engine.StartFlow(
			"wf-123", &api.ExecutionPlan{Steps: api.Steps{}}, api.Args{},
			api.Metadata{},
		)
		assert.NoError(t, err)

		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		sub := api.SubscribeRequest{
			Type: "subscribe",
			Data: api.ClientSubscription{
				AggregateID: []string{"flow", "wf-123"},
			},
		}
		err = ws.Conn.WriteJSON(sub)
		assert.NoError(t, err)

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
		var stateMsg api.SubscribedResult
		err = ws.Conn.ReadJSON(&stateMsg)
		assert.NoError(t, err)
		assert.Equal(t, []string{"flow", "wf-123"}, stateMsg.AggregateID)

		var flowState api.FlowState
		err = json.Unmarshal(stateMsg.Data, &flowState)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("wf-123"), flowState.ID)
	})
}

func TestSocketCallbackInvalidAggregate(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		for _, aggregateID := range [][]string{
			{"flow"},
			{"invalid"},
		} {
			sub := api.SubscribeRequest{
				Type: "subscribe",
				Data: api.ClientSubscription{
					AggregateID: aggregateID,
				},
			}
			err := ws.Conn.WriteJSON(sub)
			assert.NoError(t, err)

			_ = ws.Conn.SetReadDeadline(time.Now().Add(wsErrorTimeout))
			_, _, err = ws.Conn.ReadMessage()
			assert.Error(t, err)
		}
	})
}

func TestEngineEvents(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"engine"},
	}

	filter := server.BuildFilter(sub)

	engineEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("engine"),
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
		AggregateID: []string{"flow", "wf-123"},
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

func TestCombinedFiltersUseAndLogic(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"engine"},
		EventTypes:  []api.EventType{api.EventTypeStepRegistered},
	}

	filter := server.BuildFilter(sub)

	matchingEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("engine"),
	}
	wrongTypeEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("engine"),
	}
	wrongAggregateEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}

	assert.True(t, filter(matchingEvent))
	assert.False(t, filter(wrongTypeEvent))
	assert.False(t, filter(wrongAggregateEvent))
}

func TestBuildFilter(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"flow", "wf-123"},
		EventTypes:  []api.EventType{api.EventTypeFlowStarted},
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
	assert.False(t, filter(wrongFlowEvent))
}

func testWebSocket(t *testing.T, getState server.StateFunc) *testWebSocketEnv {
	t.Helper()
	hub := &mockEventHub{}

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			server.HandleWebSocket(hub, w, r, getState)
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

func testServerWebSocket(
	t *testing.T, srv *server.Server,
) *serverWebSocketEnv {
	t.Helper()

	router := srv.SetupRoutes()
	httpServer := httptest.NewServer(router)

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/engine/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	return &serverWebSocketEnv{
		Server: httpServer,
		Conn:   conn,
	}
}
