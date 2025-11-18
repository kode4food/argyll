package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/client"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestNewHTTPClient(t *testing.T) {
	timeout := 30 * time.Second
	c := client.NewHTTPClient(timeout)

	assert.NotNil(t, c)
}

func TestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Spuds-Engine/1.0", r.Header.Get("User-Agent"))

			var req api.StepRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

			response := api.StepResult{
				Success: true,
				Outputs: api.Args{
					"result": "test-output",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}
	args := api.Args{"input": "test-input"}
	meta := api.Metadata{"flow_id": "test-flow"}

	out, err := cl.Invoke(context.Background(), step, args, meta)
	require.NoError(t, err)
	assert.Equal(t, "test-output", out["result"])
}

func TestNoHTTPConfig(t *testing.T) {
	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: nil,
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	assert.Error(t, err)
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	require.Error(t, err)
	assert.Equal(t, "step returned HTTP error: HTTP 500", err.Error())
}

func TestSuccessFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			response := api.StepResult{
				Success: false,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrStepUnsuccessful)
}

func TestSuccessFalseWithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			response := api.StepResult{
				Success: false,
				Error:   "custom error message",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, client.ErrStepUnsuccessful)
	assert.Contains(t, err.Error(), "custom error message")
}

func TestInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("invalid json"))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	assert.Error(t, err)
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			response := api.StepResult{Success: true}
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(50 * time.Millisecond)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	assert.Error(t, err)
}

func TestContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			response := api.StepResult{Success: true}
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cl.Invoke(ctx, step, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestEmptyOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			response := api.StepResult{
				Success: true,
				Outputs: nil,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	outputs, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)
	assert.Nil(t, outputs)
}

func TestMultipleOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			response := api.StepResult{
				Success: true,
				Outputs: api.Args{
					"result1": "value1",
					"result2": 42,
					"result3": true,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	step := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	outputs, err := cl.Invoke(
		context.Background(), step, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)
	assert.Len(t, outputs, 3)
	assert.Equal(t, "value1", outputs["result1"])
}
