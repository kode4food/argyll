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
	step := &api.Step{
		ID:      "test-step",
		Name:    "Test Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
		},
	}

	err := client.RegisterStep(context.Background(), step)
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
	step := &api.Step{
		ID:      "test-step",
		Name:    "Test",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
		},
	}

	err := client.RegisterStep(context.Background(), step)
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
	step := &api.Step{
		ID:      "test-step",
		Name:    "Test",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
		},
	}

	err := client.RegisterStep(context.Background(), step)
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
	err := client.StartWorkflow(
		context.Background(), "wf-1", "goal-step", api.Args{"input": "value"},
	)
	assert.NoError(t, err)
}

func TestStartWithRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	req := api.CreateWorkflowRequest{
		ID:           "wf-1",
		GoalStepIDs:  []timebox.ID{"goal1", "goal2"},
		InitialState: api.Args{"input": "value"},
	}

	err := client.StartWorkflowWithRequest(context.Background(), req)
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
	err := client.StartWorkflow(
		context.Background(), "wf-1", "goal-step", api.Args{},
	)
	assert.Error(t, err)
}

func TestGetWorkflowSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/engine/workflow/wf-1", r.URL.Path)

			workflow := api.WorkflowState{
				ID:     "wf-1",
				Status: api.WorkflowActive,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(workflow)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	wf, err := client.GetWorkflow(context.Background(), "wf-1")
	require.NoError(t, err)
	assert.Equal(t, timebox.ID("wf-1"), wf.ID)
	assert.Equal(t, api.WorkflowActive, wf.Status)
}

func TestGetWorkflowError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	_, err := client.GetWorkflow(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestUpdateStateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PATCH", r.Method)
			assert.Equal(t, "/engine/workflow/wf-1/state", r.URL.Path)

			var updates api.Args
			err := json.NewDecoder(r.Body).Decode(&updates)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.UpdateState(
		context.Background(), "wf-1", api.Args{"key": "value"},
	)
	assert.NoError(t, err)
}

func TestUpdateStateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("forbidden"))
		},
	))
	defer server.Close()

	client := builder.NewClient(server.URL, 5*time.Second)
	err := client.UpdateState(context.Background(), "wf-1", api.Args{})
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

	step := &api.Step{
		ID:      "test",
		Name:    "Test",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test",
		},
	}

	err := client.RegisterStep(ctx, step)
	assert.Error(t, err)
}
