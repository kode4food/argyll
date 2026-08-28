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
			assert.Equal(t, api.JSONContentType, r.Header.Get("Content-Type"))
			assert.Equal(t, client.UserAgent, r.Header.Get("User-Agent"))

			var req api.Args
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "test-input", req["input"])
			assert.Equal(t, "test-flow", r.Header.Get(api.HeaderFlowID))

			w.Header().Set("Content-Type", api.JSONContentType)
			_ = json.NewEncoder(w).Encode(api.Args{"result": "test-output"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, api.ErrWorkNotCompleted)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.Contains(t, err.Error(), "500")
}

func TestPermanentProblem(t *testing.T) {
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
				"Content-Type", api.ProblemJSONContentType+"; charset=utf-8",
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrHTTPError)
	assert.NotContains(t, err.Error(), "validation failed")
}

func TestRetryableProblem(t *testing.T) {
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
			w.Header().Set("Content-Type", api.JSONContentType)
			_, _ = w.Write([]byte("invalid json"))
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestStepTimeoutOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", api.JSONContentType)
			_ = json.NewEncoder(w).Encode(api.Args{"result": "ok"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(50 * time.Millisecond)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: server.URL,
				Timeout:  250,
			},
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
			Invoke: api.HTTPAction{
				Endpoint: server.URL,
				Timeout:  10,
			},
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
}

func TestEmptyOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", api.JSONContentType)
			w.WriteHeader(http.StatusNoContent)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
	}

	outputs, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.NoError(t, err)
	assert.Nil(t, outputs)
}

func TestMultipleOutputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", api.JSONContentType)
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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
		ID: "test-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
		},
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

			w.Header().Set("Content-Type", api.JSONContentType)
			_ = json.NewEncoder(w).Encode(api.Args{"result": "ok"})
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID: "get-step",
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: server.URL + "/items/{item_id}",
				Method:   "GET",
			},
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
			Invoke: api.HTTPAction{
				Endpoint: "https://example.com/items/{item_id}",
				Method:   "GET",
			},
		},
	}

	_, err := cl.Invoke(st, api.Args{}, api.Metadata{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrMissingEndpointArg)
}

func TestCompensateMethod(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		},
	))
	defer server.Close()

	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:       "charge",
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: server.URL},
			Compensate: &api.HTTPAction{
				Endpoint: server.URL + "/refund/{charge_id}",
				Method:   "PUT",
			},
		},
		Attributes: api.AttributeSpecs{
			"amount":    {Role: api.RoleRequired, Compensated: true},
			"secret":    {Role: api.RoleRequired},
			"charge_id": {Role: api.RoleOutput, Compensated: true},
		},
	}

	err := cl.InvokeCompensate(
		client.CompensateRequest{
			Step:     st,
			Inputs:   api.Args{"amount": 10, "secret": "private"},
			Outputs:  api.Args{"charge_id": "ch_1"},
			Metadata: api.Metadata{},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, "PUT", gotMethod)
	assert.Equal(t, "/refund/ch_1", gotPath)

	var body api.Args
	assert.NoError(t, json.Unmarshal(gotBody, &body))
	assert.Equal(t, float64(10), body["amount"])
	assert.Equal(t, "ch_1", body["charge_id"])
	assert.NotContains(t, body, api.Name("secret"))
}

func TestCompensateConflict(t *testing.T) {
	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:       "replace",
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.com"},
			Compensate: &api.HTTPAction{
				Endpoint: "http://example.com/undo",
			},
		},
		Attributes: api.AttributeSpecs{
			"request": {
				Role:        api.RoleRequired,
				Compensated: true,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{Name: "value"},
				},
			},
			"result": {
				Role:        api.RoleOutput,
				Compensated: true,
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{Name: "value"},
				},
			},
		},
	}

	err := cl.InvokeCompensate(
		client.CompensateRequest{
			Step:    st,
			Inputs:  api.Args{"value": "before"},
			Outputs: api.Args{"value": "after"},
		},
	)
	assert.ErrorIs(t, err, api.ErrCompensateArgConflict)
}

func TestInvokeBody(t *testing.T) {
	for _, method := range []string{"GET", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			var gotBody []byte
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, method, r.Method)
					assert.Empty(t, r.Header.Get("Content-Type"))
					gotBody, _ = io.ReadAll(r.Body)
					w.WriteHeader(http.StatusNoContent)
				},
			))
			defer server.Close()

			cl := client.NewHTTPClient(5 * time.Second)
			st := &api.Step{
				ID: "step",
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: server.URL,
						Method:   method,
					},
				},
			}

			_, err := cl.Invoke(st, api.Args{"value": "x"}, nil)
			assert.NoError(t, err)
			assert.Empty(t, gotBody)
		})
	}
}

func TestCompensateBody(t *testing.T) {
	for _, method := range []string{"GET", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			var gotBody []byte
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, method, r.Method)
					assert.Empty(t, r.Header.Get("Content-Type"))
					gotBody, _ = io.ReadAll(r.Body)
					w.WriteHeader(http.StatusNoContent)
				},
			))
			defer server.Close()

			cl := client.NewHTTPClient(5 * time.Second)
			st := &api.Step{
				ID: "step",
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{Endpoint: server.URL},
					Compensate: &api.HTTPAction{
						Endpoint: server.URL + "/undo",
						Method:   method,
					},
				},
			}

			err := cl.InvokeCompensate(client.CompensateRequest{Step: st})
			assert.NoError(t, err)
			assert.Empty(t, gotBody)
		})
	}
}

func TestCompensateTimeout(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
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
		ID: "step",
		HTTP: &api.HTTPConfig{
			Invoke:     api.HTTPAction{Endpoint: server.URL, Timeout: 10},
			Compensate: &api.HTTPAction{Endpoint: server.URL + "/undo"},
		},
	}

	err := cl.InvokeCompensate(client.CompensateRequest{Step: st})
	assert.Error(t, err)
}

func TestCompensateMissing(t *testing.T) {
	cl := client.NewHTTPClient(5 * time.Second)
	st := &api.Step{
		ID:   "step",
		HTTP: &api.HTTPConfig{Invoke: api.HTTPAction{Endpoint: "http://x"}},
	}

	err := cl.InvokeCompensate(client.CompensateRequest{Step: st})
	assert.ErrorIs(t, err, client.ErrNoHTTPConfig)
}
