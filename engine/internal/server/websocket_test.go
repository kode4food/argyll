package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	testWebSocketEnv struct {
		Server *httptest.Server
		Env    *helpers.TestEngineEnv
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

func (e *testWebSocketEnv) Cleanup() {
	if e.Conn != nil {
		_ = e.Conn.Close()
	}
	if e.Server != nil {
		e.Server.Close()
	}
	if e.Env != nil {
		e.Env.Cleanup()
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
	flowID := api.FlowID("wf-123")

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
		Type: timebox.EventType(api.EventTypeFlowStarted),
		Data: json.RawMessage(`{"test":"data"}`),
	}
	err = env.Env.AppendFlowEvents(flowID, event)
	assert.NoError(t, err)

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
	flowID := api.FlowID("wf-123")

	err := env.Conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	assert.NoError(t, err)

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
		Data: json.RawMessage(`{}`),
	}
	err = env.Env.AppendFlowEvents(flowID, event)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.Error(t, err)
}

func TestMessageNonSubscribe(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()
	flowID := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "other",
		Data: api.ClientSubscription{
			AggregateID: []string{"flow", "wf-123"},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
		Data: json.RawMessage(`{}`),
	}
	err = env.Env.AppendFlowEvents(flowID, event)
	assert.NoError(t, err)

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
		return &api.FlowState{ID: "wf-123"}, 1, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()
	flowID := api.FlowID("wf-123")

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
	assert.Equal(t, int64(1), stateMsg.Sequence)

	// Send stale event (sequence 5 < minSequence 10)
	staleEvent := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
		Data: json.RawMessage(`{"stale":true}`),
	}
	err = env.Env.AppendFlowEvents(flowID, staleEvent)
	assert.NoError(t, err)

	// Send fresh event (sequence 10 >= minSequence 10)
	freshEvent := &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepStarted),
		Data: json.RawMessage(`{"fresh":true}`),
	}
	err = env.Env.AppendFlowEvents(flowID, freshEvent)
	assert.NoError(t, err)

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

	_ = env.Conn.Close()

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

func TestSocketCallbackInvalidAgg(t *testing.T) {
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

func testWebSocket(t *testing.T, getState server.StateFunc) *testWebSocketEnv {
	t.Helper()
	env := helpers.NewTestEngine(t)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			server.HandleWebSocket(env.EventHub, w, r, getState)
		},
	))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	return &testWebSocketEnv{
		Server: srv,
		Env:    env,
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
