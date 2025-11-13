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

func TestNewAsyncContext(t *testing.T) {
	meta := api.Metadata{
		"flow_id":     "test-flow",
		"step_id":     "test-step",
		"webhook_url": "http://localhost:8080/webhook/test-flow/test-step/t123",
	}
	ctx := context.WithValue(context.Background(), builder.MetadataKey, meta)

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	ac, err := client.NewAsyncContext(ctx)
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
			_, err := client.NewAsyncContext(ctx)
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ac, err := client.NewAsyncContext(ctx)
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ac, err := client.NewAsyncContext(ctx)
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ac, err := client.NewAsyncContext(ctx)
	require.NoError(t, err)

	err = ac.Success(api.Args{"key": "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status 500")
}
