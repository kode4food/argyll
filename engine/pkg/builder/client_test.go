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
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "registered",
			})
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)

	err := client.NewStep("Test Step").
		WithEndpoint("http://test").
		Register(context.Background())

	assert.NoError(t, err)
}

func TestRegisterStepStatusCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)

	err := client.NewStep("Test").
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

	err := client.NewStep("Test").
		WithEndpoint("http://test").
		Register(context.Background())

	assert.Error(t, err)
}

func TestStartSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/engine/workflow", r.URL.Path)

			var req api.CreateWorkflowRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	ctx := context.Background()
	err := client.NewWorkflow("wf-1").
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
	err := client.NewWorkflow("wf-1").
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
	require.NoError(t, err)
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
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.NewStep("Test").
		WithEndpoint("http://test").
		Register(ctx)

	assert.Error(t, err)
}

func TestWorkflow(t *testing.T) {
	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	wc := client.Workflow("test-flow-123")
	assert.Equal(t, timebox.ID("test-flow-123"), wc.FlowID())
}

func TestWorkflowGetState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/engine/workflow/my-flow", r.URL.Path)

			workflow := api.WorkflowState{
				ID:     "my-flow",
				Status: api.WorkflowActive,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(workflow)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wc := client.Workflow("my-flow")

	state, err := wc.GetState(context.Background())
	require.NoError(t, err)
	assert.Equal(t, timebox.ID("my-flow"), state.ID)
	assert.Equal(t, api.WorkflowActive, state.Status)
}
