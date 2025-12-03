package builder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}

	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-flow", ac.FlowID())
	assert.Equal(t, "test-step", ac.StepID())
	assert.Equal(t,
		"http://localhost:8080/webhook/test-flow/test-step/t123",
		ac.WebhookURL(),
	)
}

func TestAsyncContextMissingMeta(t *testing.T) {
	tests := []struct {
		name    string
		meta    api.Metadata
		wantErr error
	}{
		{
			name:    "no metadata",
			meta:    nil,
			wantErr: builder.ErrMetadataNotFound,
		},
		{
			name:    "missing webhook_url",
			meta:    api.Metadata{},
			wantErr: builder.ErrWebhookURLNotFound,
		},
	}

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &builder.StepContext{
				Context:  context.Background(),
				Client:   client.Flow("test-flow"),
				StepID:   "test-step",
				Metadata: tt.meta,
			}
			_, err := builder.NewAsyncContext(ctx)
			assert.ErrorIs(t, err, tt.wantErr)
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
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

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
	ac, err := builder.NewAsyncContext(ctx)
	require.NoError(t, err)

	err = ac.Success(api.Args{"key": "value"})
	assert.ErrorIs(t, err, builder.ErrWebhookError)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "internal error")
}
