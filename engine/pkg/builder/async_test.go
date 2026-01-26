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
	"github.com/kode4food/argyll/engine/pkg/builder"
)

func TestNewAsyncContext(t *testing.T) {
	meta := api.Metadata{
		api.MetaFlowID:     "test-flow",
		api.MetaStepID:     "test-step",
		api.MetaWebhookURL: "http://localhost:8080/webhook/" +
			"test-flow/test-step/t123",
	}

	client := builder.NewClient("http://localhost:8080", 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}

	ac, err := builder.NewAsyncContext(ctx)
	assert.NoError(t, err)
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
			assert.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, "result-value", result.Outputs["output_key"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	meta := api.Metadata{
		api.MetaFlowID:     "test-flow",
		api.MetaStepID:     "test-step",
		api.MetaWebhookURL: server.URL,
	}

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
	ac, err := builder.NewAsyncContext(ctx)
	assert.NoError(t, err)

	err = ac.Success(api.Args{"output_key": "result-value"})
	assert.NoError(t, err)
}

func TestAsyncContextFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var result api.StepResult
			err := json.NewDecoder(r.Body).Decode(&result)
			assert.NoError(t, err)
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "general error")

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	meta := api.Metadata{
		api.MetaFlowID:     "test-flow",
		api.MetaStepID:     "test-step",
		api.MetaWebhookURL: server.URL,
	}

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
	ac, err := builder.NewAsyncContext(ctx)
	assert.NoError(t, err)

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
		api.MetaFlowID:     "test-flow",
		api.MetaStepID:     "test-step",
		api.MetaWebhookURL: server.URL,
	}

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
	ac, err := builder.NewAsyncContext(ctx)
	assert.NoError(t, err)

	err = ac.Success(api.Args{"key": "value"})
	assert.ErrorIs(t, err, builder.ErrWebhookError)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "internal error")
}

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			var result api.StepResult
			err := json.NewDecoder(r.Body).Decode(&result)
			assert.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, "value1", result.Outputs["key1"])
			assert.Equal(t, "value2", result.Outputs["key2"])

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	meta := api.Metadata{
		api.MetaFlowID:     "test-flow",
		api.MetaStepID:     "test-step",
		api.MetaWebhookURL: server.URL,
	}

	client := builder.NewClient(server.URL, 30*time.Second)
	ctx := &builder.StepContext{
		Context:  context.Background(),
		Client:   client.Flow("test-flow"),
		StepID:   "test-step",
		Metadata: meta,
	}
	ac, err := builder.NewAsyncContext(ctx)
	assert.NoError(t, err)

	result := api.StepResult{
		Success: true,
		Outputs: api.Args{
			"key1": "value1",
			"key2": "value2",
		},
	}
	err = ac.Complete(result)
	assert.NoError(t, err)
}
