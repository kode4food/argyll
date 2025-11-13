package builder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

func TestNewWorkflow(t *testing.T) {
	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	flowID := timebox.ID("test-workflow")

	wf := client.NewWorkflow(flowID)

	assert.NotNil(t, wf)
}

func TestWorkflowWithGoals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, timebox.ID("wf-1"), req.ID)
			assert.Len(t, req.Goals, 2)
			assert.Equal(t, timebox.ID("goal-1"), req.Goals[0])
			assert.Equal(t, timebox.ID("goal-2"), req.Goals[1])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoals("goal-1", "goal-2").
		Start(context.Background())

	assert.NoError(t, err)
}

func TestWorkflowWithGoal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, timebox.ID("wf-1"), req.ID)
			assert.Len(t, req.Goals, 3)
			assert.Equal(t, timebox.ID("goal-1"), req.Goals[0])
			assert.Equal(t, timebox.ID("goal-2"), req.Goals[1])
			assert.Equal(t, timebox.ID("goal-3"), req.Goals[2])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoal("goal-1").
		WithGoal("goal-2").
		WithGoal("goal-3").
		Start(context.Background())

	assert.NoError(t, err)
}

func TestWorkflowWithInitialState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, timebox.ID("wf-1"), req.ID)
			assert.Equal(t, "value1", req.Init["key1"])
			assert.Equal(t, float64(42), req.Init["key2"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoals("goal-step").
		WithInitialState(api.Args{
			"key1": "value1",
			"key2": 42,
		}).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestWorkflowStartStatusCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoals("goal-step").
		Start(context.Background())

	assert.NoError(t, err)
}

func TestWorkflowStartError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoals("goal-step").
		Start(context.Background())

	assert.Error(t, err)
}

func TestWorkflowChaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, timebox.ID("complex-workflow"), req.ID)
			assert.Len(t, req.Goals, 2)
			assert.Equal(t, timebox.ID("goal-1"), req.Goals[0])
			assert.Equal(t, timebox.ID("goal-2"), req.Goals[1])
			assert.Equal(t, "value1", req.Init["arg1"])
			assert.Equal(t, float64(100), req.Init["arg2"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("complex-workflow").
		WithGoals("goal-1", "goal-2").
		WithInitialState(api.Args{
			"arg1": "value1",
			"arg2": 100,
		}).
		Start(context.Background())

	assert.NoError(t, err)
}

func TestWorkflowImmutability(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Len(t, req.Goals, 1)
			assert.Equal(t, timebox.ID("goal-1"), req.Goals[0])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Len(t, req.Goals, 1)
			assert.Equal(t, timebox.ID("goal-2"), req.Goals[0])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server2.Close()

	client1 := builder.NewClient(server1.URL, 5*time.Second)
	err1 := client1.NewWorkflow("base-wf").
		WithGoals("goal-1").
		Start(context.Background())
	assert.NoError(t, err1)

	client2 := builder.NewClient(server2.URL, 5*time.Second)
	err2 := client2.NewWorkflow("base-wf").
		WithGoals("goal-2").
		Start(context.Background())
	assert.NoError(t, err2)
}

func TestWorkflowImmutabilityInitState(t *testing.T) {
	initState1 := api.Args{"key": "value1"}
	initState2 := api.Args{"key": "value2"}

	server1 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "value1", req.Init["key"])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "value2", req.Init["key"])
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server2.Close()

	client1 := builder.NewClient(server1.URL, 5*time.Second)
	err1 := client1.NewWorkflow("test-wf").
		WithGoals("goal").
		WithInitialState(initState1).
		Start(context.Background())
	assert.NoError(t, err1)

	client2 := builder.NewClient(server2.URL, 5*time.Second)
	err2 := client2.NewWorkflow("test-wf").
		WithGoals("goal").
		WithInitialState(initState2).
		Start(context.Background())
	assert.NoError(t, err2)
}

func TestWorkflowEmptyGoals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Len(t, req.Goals, 0)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").Start(context.Background())
	assert.NoError(t, err)
}

func TestWorkflowEmptyInitialState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Empty(t, req.Init)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewWorkflow("wf-1").
		WithGoals("goal-step").
		Start(context.Background())
	assert.NoError(t, err)
}
