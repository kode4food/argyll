package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/pkg/api"
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
			assert.Equal(t, "Argyll-Engine/1.0", r.Header.Get("User-Agent"))

			var req api.Args
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "test-input", req["input"])
			assert.Equal(t, "test-flow", r.Header.Get(api.HeaderFlowID))

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.Args{"result": "test-output"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}
	args := api.Args{"input": "test-input"}
	meta := api.Metadata{api.MetaFlowID: "test-flow"}

	out, err := cl.Invoke(st, args, meta)
	assert.NoError(t, err)
	assert.Equal(t, "test-output", out["result"])
}

func TestNoHTTPConfig(t *testing.T) {
	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: nil,
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
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
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, api.ErrWorkNotCompleted)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.Contains(t, err.Error(), "500")
}

func TestProblemDetailsPermanentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", api.ProblemJSONContentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(api.NewProblem(
				http.StatusUnprocessableEntity, "validation failed",
			))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.NotErrorIs(t, err, api.ErrWorkNotCompleted)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestProblemMediaParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(
				"Content-Type",
				api.ProblemJSONContentType+"; charset=utf-8",
			)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(api.NewProblem(
				http.StatusUnprocessableEntity, "validation failed",
			))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestProblemMediaRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", api.JSONContentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(api.NewProblem(
				http.StatusUnprocessableEntity, "validation failed",
			))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.NotContains(t, err.Error(), "validation failed")
}

func TestProblemDetailsRetryableFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", api.ProblemJSONContentType)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(api.NewProblem(
				http.StatusServiceUnavailable, "custom error message",
			))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, api.ErrWorkNotCompleted)
	assert.ErrorIs(t, err, client.ErrHTTPError)
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
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestTimeout(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-serverDone:
			}
		},
	))
	defer server.Close()
	defer close(serverDone)

	cl := client.NewHTTPClient(50 * time.Millisecond)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestStepTimeoutOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.Args{"result": "ok"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(50 * time.Millisecond)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Endpoint: server.URL,
			Timeout:  250,
		},
	}

	outputs, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.NoError(t, err)
	assert.Equal(t, "ok", outputs["result"])
}

func TestStepTimeoutShorter(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-serverDone:
			}
		},
	))
	defer server.Close()
	defer close(serverDone)

	cl := client.NewHTTPClient(1 * time.Second)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Endpoint: server.URL,
			Timeout:  10,
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestEmptyOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	outputs, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.NoError(t, err)
	assert.Nil(t, outputs)
}

func TestMultipleOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.Args{
				"result1": "value1",
				"result2": 42,
				"result3": true,
			})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	outputs, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.NoError(t, err)
	assert.Len(t, outputs, 3)
	assert.Equal(t, "value1", outputs["result1"])
}

func TestHTTP4xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "test-step",
		HTTP: &api.HTTPConfig{Endpoint: server.URL},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.NotErrorIs(t, err, api.ErrWorkNotCompleted)
	assert.Contains(t, err.Error(), "400")
}

func TestGETURLParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/items/abc%20123", r.URL.EscapedPath())
			assert.Empty(t, r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Empty(t, body)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.Args{"result": "ok"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "get-step",
		HTTP: &api.HTTPConfig{
			Endpoint: server.URL + "/items/{item_id}",
			Method:   "GET",
		},
	}

	outputs, err := cl.Invoke(
		st, api.Args{"item_id": "abc 123"}, api.Metadata{},
	)
	assert.NoError(t, err)
	assert.Equal(t, "ok", outputs["result"])
}

func TestMissingURLArg(t *testing.T) {
	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "missing-arg-step",
		HTTP: &api.HTTPConfig{
			Endpoint: "https://example.com/items/{item_id}",
			Method:   "GET",
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrMissingEndpointArg)
}
