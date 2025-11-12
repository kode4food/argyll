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
	ctx := context.Background()
	err := client.StartWorkflow(
		ctx, "wf-1", []timebox.ID{"goal-step"}, api.Args{"input": "value"},
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
		ID:    "wf-1",
		Goals: []timebox.ID{"goal1", "goal2"},
		Init:  api.Args{"input": "value"},
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
		context.Background(), "wf-1", []timebox.ID{"goal-step"}, api.Args{},
	)
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

func TestWorkflow(t *testing.T) {
	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	wc := client.Workflow("test-flow-123")
	assert.Equal(t, timebox.ID("test-flow-123"), wc.FlowID())
}

func TestWorkflowFromCtx(t *testing.T) {
	meta := api.Metadata{
		"flow_id": "test-flow-789",
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	wc, err := client.WorkflowFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, timebox.ID("test-flow-789"), wc.FlowID())
}

func TestWorkflowFromCtxMissing(t *testing.T) {
	tests := []struct {
		name     string
		meta     api.Metadata
		errMatch string
	}{
		{
			name:     "no metadata",
			meta:     nil,
			errMatch: "metadata not found",
		},
		{
			name:     "missing flow_id",
			meta:     api.Metadata{"other_field": "value"},
			errMatch: "flow_id not found",
		},
	}

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.meta != nil {
				ctx = context.WithValue(
					context.Background(), builder.MetadataKey, tt.meta,
				)
			} else {
				ctx = context.Background()
			}
			_, err := client.WorkflowFromContext(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMatch)
		})
	}
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

func TestNewAsyncContext(t *testing.T) {
	meta := api.Metadata{
		"flow_id":     "test-flow",
		"step_id":     "test-step",
		"webhook_url": "http://localhost:8080/webhook/test-flow/test-step/t123",
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, timebox.ID("test-flow"), ac.FlowID())
	assert.Equal(t, timebox.ID("test-step"), ac.StepID())
	assert.Equal(t,
		"http://localhost:8080/webhook/test-flow/test-step/t123",
		ac.WebhookURL(),
	)
}

func TestAsyncContextMissingMeta(t *testing.T) {
	tests := []struct {
		name     string
		meta     api.Metadata
		errMatch string
	}{
		{
			name:     "no metadata",
			meta:     nil,
			errMatch: "metadata not found",
		},
		{
			name: "missing flow_id",
			meta: api.Metadata{
				"step_id":     "step",
				"webhook_url": "http://test",
			},
			errMatch: "flow_id not found",
		},
		{
			name: "missing step_id",
			meta: api.Metadata{
				"flow_id":     "flow",
				"webhook_url": "http://test",
			},
			errMatch: "step_id not found",
		},
		{
			name:     "missing webhook_url",
			meta:     api.Metadata{"flow_id": "flow", "step_id": "step"},
			errMatch: "webhook_url not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.meta != nil {
				ctx = context.WithValue(
					context.Background(), builder.MetadataKey, tt.meta,
				)
			} else {
				ctx = context.Background()
			}
			_, err := builder.NewAsyncContext(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMatch)
		})
	}
}

func TestAsyncContextComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			var result api.StepResult
			err := json.NewDecoder(r.Body).Decode(&result)
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, "result-value", result.Outputs["output_key"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	meta := api.Metadata{
		"flow_id":     "test-flow",
		"step_id":     "test-step",
		"webhook_url": server.URL,
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)

	err = ac.Success(api.Args{"output_key": "result-value"})
	assert.NoError(t, err)
}

func TestAsyncContextFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var result api.StepResult
			err := json.NewDecoder(r.Body).Decode(&result)
			require.NoError(t, err)
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "general error")

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	meta := api.Metadata{
		"flow_id":     "test-flow",
		"step_id":     "test-step",
		"webhook_url": server.URL,
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)

	err = ac.Fail(assert.AnError)
	assert.NoError(t, err)
}

func TestAsyncContextWebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		},
	))
	defer server.Close()

	meta := api.Metadata{
		"flow_id":     "test-flow",
		"step_id":     "test-step",
		"webhook_url": server.URL,
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)

	err = ac.Success(api.Args{"key": "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status 500")
}
