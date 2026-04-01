package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
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

func subscribedItem(
	t *testing.T, msg api.SubscribedResult,
) api.SubscribedItem {
	t.Helper()
	assert.Len(t, msg.Items, 1)
	return msg.Items[0]
}

func TestClientReceivesEvent(t *testing.T) {
	getState := func(aggID timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)

	assert.Equal(t, api.EventTypeFlowStarted, wsEvent.Type)
	var data api.FlowStartedEvent
	err = json.Unmarshal(wsEvent.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, id, data.FlowID)
}

func TestClientFiltersEventsByType(t *testing.T) {
	getState := func(aggID timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
			EventTypes:     []api.EventType{api.EventTypeFlowStarted},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeStepStarted,
		Data: wsStepStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, api.EventTypeFlowStarted, wsEvent.Type)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestClientAggregateEvents(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		assert.Fail(t, "multi-aggregate subscribe should not load state")
		return nil, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs: [][]string{
				{events.FlowPrefix, "wf-123"},
				{events.FlowPrefix, "wf-456"},
			},
			IncludeState: false,
			EventTypes:   []api.EventType{api.EventTypeFlowStarted},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var ack api.SubscribedResult
	err = env.Conn.ReadJSON(&ack)
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents("wf-123", helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted("wf-123", "step-1"),
	})
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents("wf-456", helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted("wf-456", "step-1"),
	})
	assert.NoError(t, err)

	var wsEvent api.WebSocketEvent
	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, []string{events.FlowPrefix, "wf-123"}, wsEvent.AggregateID)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, []string{events.FlowPrefix, "wf-456"}, wsEvent.AggregateID)
}

func TestClientManyAggregates(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		assert.Fail(t, "multi-aggregate subscribe should not load state")
		return nil, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	aggregateIDs := make([][]string, 0, 200)
	for i := range 200 {
		aggregateIDs = append(aggregateIDs,
			[]string{events.FlowPrefix, "wf-" + strconv.Itoa(i)})
	}

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   aggregateIDs,
			IncludeState:   false,
			EventTypes:     []api.EventType{api.EventTypeFlowStarted},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var ack api.SubscribedResult
	err = env.Conn.ReadJSON(&ack)
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents("wf-199", helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted("wf-199", "step-1"),
	})
	assert.NoError(t, err)

	var wsEvent api.WebSocketEvent
	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, []string{events.FlowPrefix, "wf-199"}, wsEvent.AggregateID)
}

func TestMessageInvalid(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	err := env.Conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.Error(t, err)
}

func TestMessageNonSubscribe(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "other",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.Error(t, err)
}

func TestSubscribeStateSendsState(t *testing.T) {
	fl := &api.FlowState{
		ID:     "wf-123",
		Status: api.FlowActive,
	}

	getState := func(id timebox.AggregateID) (any, int64, error) {
		assert.Equal(t, events.FlowKey("wf-123"), id)
		return fl, 5, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	assert.Equal(t, "subscribed", stateMsg.Type)
	item := subscribedItem(t, stateMsg)
	assert.Equal(t, []string{events.FlowPrefix, "wf-123"}, item.AggregateID)
	assert.Equal(t, int64(5), item.Sequence)

	var receivedState api.FlowState
	err = json.Unmarshal(item.Data, &receivedState)
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("wf-123"), receivedState.ID)
	assert.Equal(t, api.FlowActive, receivedState.Status)
}

func TestStaleEventsFiltered(t *testing.T) {
	getState := func(aggID timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 1, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	item := subscribedItem(t, stateMsg)
	assert.Equal(t, int64(1), item.Sequence)

	// Send stale event (sequence 5 < minSequence 10)
	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	// Send fresh event (sequence 10 >= minSequence 10)
	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeStepStarted,
		Data: wsStepStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	// Should only receive the fresh event
	var wsEvent api.WebSocketEvent
	err = env.Conn.ReadJSON(&wsEvent)
	assert.NoError(t, err)
	assert.Equal(t, api.EventTypeStepStarted, wsEvent.Type)
	var stepData api.StepStartedEvent
	err = json.Unmarshal(wsEvent.Data, &stepData)
	assert.NoError(t, err)
	assert.Equal(t, api.StepID("step-1"), stepData.StepID)
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
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestSubscribeStateMissingFlow(t *testing.T) {
	getState := func(id timebox.AggregateID) (any, int64, error) {
		return nil, 0, engine.ErrFlowNotFound
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-404"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg api.SubscribedResult
	err = env.Conn.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "subscribed", msg.Type)
	assert.Empty(t, msg.Items)

	err = env.Conn.WriteJSON(api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-2",
			AggregateIDs:   [][]string{{events.CatalogPrefix}},
		},
	})
	assert.NoError(t, err)
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
			SubscriptionID: "sub-1",
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
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err = env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)
	item := subscribedItem(t, stateMsg)
	assert.Equal(t, []string{events.FlowPrefix, "wf-123"}, item.AggregateID)
}

func TestClientUnsubscribeStopsEvents(t *testing.T) {
	getState := func(aggID timebox.AggregateID) (any, int64, error) {
		return &api.FlowState{ID: "wf-123"}, 0, nil
	}

	env := testWebSocket(t, getState)
	defer env.Cleanup()
	id := api.FlowID("wf-123")

	sub := api.SubscribeRequest{
		Type: "subscribe",
		Data: api.ClientSubscription{
			SubscriptionID: "sub-1",
			AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
			IncludeState:   true,
		},
	}
	err := env.Conn.WriteJSON(sub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
	var stateMsg api.SubscribedResult
	err = env.Conn.ReadJSON(&stateMsg)
	assert.NoError(t, err)

	unsub := api.UnsubscribeRequest{
		Type: "unsubscribe",
		Data: api.ClientUnsubscription{
			SubscriptionID: "sub-1",
		},
	}
	err = env.Conn.WriteJSON(unsub)
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	var unsubAck api.UnsubscribedResult
	err = env.Conn.ReadJSON(&unsubAck)
	assert.NoError(t, err)
	assert.Equal(t, "unsubscribed", unsubAck.Type)

	err = env.Env.RaiseFlowEvents(id, helpers.FlowEvent{
		Type: api.EventTypeFlowStarted,
		Data: wsFlowStarted(id, "step-1"),
	})
	assert.NoError(t, err)

	_ = env.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestClientConsumerClosed(t *testing.T) {
	env := testWebSocket(t, nil)
	defer env.Cleanup()

	_ = env.Conn.Close()

	_ = env.Conn.SetReadDeadline(time.Now().Add(wsCloseTimeout))
	_, _, err := env.Conn.ReadMessage()
	assert.Error(t, err)
}

func TestServerCloseWebSockets(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		env.Server.CloseWebSockets()

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsCloseTimeout))
		_, _, err := ws.Conn.ReadMessage()
		assert.Error(t, err)
	})
}

func TestSocketCallbackEngine(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		sub := api.SubscribeRequest{
			Type: "subscribe",
			Data: api.ClientSubscription{
				SubscriptionID: "sub-1",
				AggregateIDs:   [][]string{{events.CatalogPrefix}},
				IncludeState:   true,
			},
		}
		err := ws.Conn.WriteJSON(sub)
		assert.NoError(t, err)

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
		var stateMsg api.SubscribedResult
		err = ws.Conn.ReadJSON(&stateMsg)
		assert.NoError(t, err)
		item := subscribedItem(t, stateMsg)
		assert.Equal(t, []string{events.CatalogPrefix}, item.AggregateID)

		var cat api.CatalogState
		err = json.Unmarshal(item.Data, &cat)
		assert.NoError(t, err)
	})
}

func TestSocketCallbackCluster(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		sub := api.SubscribeRequest{
			Type: "subscribe",
			Data: api.ClientSubscription{
				SubscriptionID: "sub-1",
				AggregateIDs:   [][]string{{events.ClusterPrefix}},
				IncludeState:   true,
			},
		}
		err := ws.Conn.WriteJSON(sub)
		assert.NoError(t, err)

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
		var stateMsg api.SubscribedResult
		err = ws.Conn.ReadJSON(&stateMsg)
		assert.NoError(t, err)
		item := subscribedItem(t, stateMsg)
		assert.Equal(t, []string{events.ClusterPrefix}, item.AggregateID)

		var cluster api.ClusterState
		err = json.Unmarshal(item.Data, &cluster)
		assert.NoError(t, err)
	})
}

func TestSocketCallbackFlow(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		err := env.Engine.StartFlow("wf-123", &api.ExecutionPlan{
			Steps: api.Steps{},
		})
		assert.NoError(t, err)

		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		sub := api.SubscribeRequest{
			Type: "subscribe",
			Data: api.ClientSubscription{
				SubscriptionID: "sub-1",
				AggregateIDs:   [][]string{{events.FlowPrefix, "wf-123"}},
				IncludeState:   true,
			},
		}
		err = ws.Conn.WriteJSON(sub)
		assert.NoError(t, err)

		_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
		var stateMsg api.SubscribedResult
		err = ws.Conn.ReadJSON(&stateMsg)
		assert.NoError(t, err)
		item := subscribedItem(t, stateMsg)
		assert.Equal(t, []string{events.FlowPrefix, "wf-123"}, item.AggregateID)

		var flow api.FlowState
		err = json.Unmarshal(item.Data, &flow)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("wf-123"), flow.ID)
	})
}

func TestSocketBadCallbackTarget(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		ws := testServerWebSocket(t, env.Server)
		defer ws.Cleanup()

		for _, aggregateID := range [][]string{
			{events.FlowPrefix},
			{"invalid"},
		} {
			sub := api.SubscribeRequest{
				Type: "subscribe",
				Data: api.ClientSubscription{
					SubscriptionID: "sub-1",
					AggregateIDs:   [][]string{aggregateID},
					IncludeState:   true,
				},
			}
			err := ws.Conn.WriteJSON(sub)
			assert.NoError(t, err)

			_ = ws.Conn.SetReadDeadline(time.Now().Add(wsStateTimeout))
			var stateMsg api.SubscribedResult
			err = ws.Conn.ReadJSON(&stateMsg)
			assert.NoError(t, err)
			item := subscribedItem(t, stateMsg)
			assert.Equal(t, aggregateID, item.AggregateID)
			assert.Equal(t, json.RawMessage("null"), item.Data)
		}
	})
}

func testWebSocket(t *testing.T, getState server.StateFunc) *testWebSocketEnv {
	t.Helper()
	env := helpers.NewTestEngine(t)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			server.HandleWebSocket(env.EventHub, w, r, getState, nil)
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

func wsFlowStarted(
	flowID api.FlowID, stepID api.StepID,
) api.FlowStartedEvent {
	return api.FlowStartedEvent{
		FlowID:   flowID,
		Plan:     wsPlan(stepID),
		Init:     api.Args{},
		Metadata: api.Metadata{},
	}
}

func wsStepStarted(
	flowID api.FlowID, stepID api.StepID,
) api.StepStartedEvent {
	return api.StepStartedEvent{
		FlowID:    flowID,
		StepID:    stepID,
		Inputs:    api.Args{},
		WorkItems: map[api.Token]api.Args{},
	}
}

func wsPlan(stepID api.StepID) *api.ExecutionPlan {
	step := &api.Step{
		ID:   stepID,
		Name: "ws-step",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}
	return &api.ExecutionPlan{
		Goals: []api.StepID{stepID},
		Steps: api.Steps{stepID: step},
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
