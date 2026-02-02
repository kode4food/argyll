package builder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdks/go-builder"
)

func TestNewClient(t *testing.T) {
	timeout := 30 * time.Second
	client := builder.NewClient("http://localhost:8080", timeout)

	assert.NotNil(t, client)
}

func TestRegisterStepSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/engine/step", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var step api.Step
			err := json.NewDecoder(r.Body).Decode(&step)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "registered",
			})
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)

	err := client.NewStep().WithName("Test Step").
		WithEndpoint("http://test").
		Register(context.Background())

	assert.NoError(t, err)
}

func TestRegisterCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)

	err := client.NewStep().WithName("Test").
		WithEndpoint("http://test").
		Register(context.Background())

	assert.NoError(t, err)
}

func TestRegisterStepError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)

	err := client.NewStep().WithName("Test").
		WithEndpoint("http://test").
		Register(context.Background())

	assert.Error(t, err)
}

func TestStartSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/engine/flow", r.URL.Path)

			var req api.CreateFlowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	ctx := context.Background()
	err := client.NewFlow("wf-1").
		WithGoals("goal-step").
		WithInitialState(api.Args{"input": "value"}).
		Start(ctx)
	assert.NoError(t, err)
}

func TestStartError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.NewFlow("wf-1").
		WithGoals("goal-step").
		Start(context.Background())
	assert.Error(t, err)
}

func TestListStepsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/engine/step", r.URL.Path)

			response := api.StepsListResponse{
				Steps: []*api.Step{
					{ID: "step1", Name: "Step 1"},
					{ID: "step2", Name: "Step 2"},
				},
				Count: 2,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	result, err := client.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Count)
	assert.Len(t, result.Steps, 2)
}

func TestListStepsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	_, err := client.ListSteps(context.Background())
	assert.Error(t, err)
}

func TestContextCancellation(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-serverDone:
			}
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()
	defer close(serverDone)

	client := builder.NewClient(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.NewStep().WithName("Test").
		WithEndpoint("http://test").
		Register(ctx)

	assert.Error(t, err)
}

func TestFlow(t *testing.T) {
	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	wc := client.Flow("test-flow-123")
	assert.Equal(t, api.FlowID("test-flow-123"), wc.FlowID())
}

func TestFlowGetState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/engine/flow/my-flow", r.URL.Path)

			flow := api.FlowState{
				ID:     "my-flow",
				Status: api.FlowActive,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(flow)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Flow("my-flow")

	state, err := wc.GetState(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("my-flow"), state.ID)
	assert.Equal(t, api.FlowActive, state.Status)
}

func TestGetState404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("flow not found"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Flow("nonexistent-flow")

	_, err := wc.GetState(context.Background())
	assert.Error(t, err)
}

func TestGetState500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal server error"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Flow("test-flow")

	_, err := wc.GetState(context.Background())
	assert.Error(t, err)
}

func TestGetStateMalformed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not valid json{"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Flow("test-flow")

	_, err := wc.GetState(context.Background())
	assert.Error(t, err)
}

func TestGetStateCancelled(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-serverDone:
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(api.FlowState{})
		},
	))
	defer server.Close()
	defer close(serverDone)

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Flow("test-flow")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wc.GetState(ctx)
	assert.Error(t, err)
}

func TestGetStateNetworkError(t *testing.T) {
	client := builder.NewClient("http://localhost:1", 1*time.Millisecond)
	wc := client.Flow("test-flow")

	_, err := wc.GetState(context.Background())
	assert.Error(t, err)
}
