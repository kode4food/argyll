package builder_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdks/go-builder"
)

func TestHTTPError(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return nil, builder.NewHTTPError(http.StatusTeapot, "teapot")
	}

	stepURL := startStepServer(t,
		engineServer.URL, "test-step", "test-step", handler,
	)

	body, err := json.Marshal(api.Args{"foo": "bar"})
	assert.NoError(t, err)

	resp, err := http.Post(stepURL, api.JSONContentType, bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
}

func TestStepHandlerRejectsBadRequests(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return api.Args{}, nil
	}

	stepURL := startStepServer(t,
		engineServer.URL, "bad-step", "bad-step", handler,
	)

	resp, err := http.Get(stepURL)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	resp, err = http.Post(
		stepURL, api.JSONContentType, bytes.NewBufferString("{"),
	)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStepHandlerPlainError(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return nil, assert.AnError
	}

	stepURL := startStepServer(t,
		engineServer.URL, "error-step", "error-step", handler,
	)

	body, err := json.Marshal(api.Args{})
	assert.NoError(t, err)

	resp, err := http.Post(stepURL, api.JSONContentType, bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestPanic(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		panic("boom")
	}

	stepURL := startStepServer(t,
		engineServer.URL, "panic-step", "panic-step", handler,
	)

	body, err := json.Marshal(api.Args{})
	assert.NoError(t, err)

	resp, err := http.Post(stepURL, api.JSONContentType, bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var problem api.ProblemDetails
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Contains(t, problem.Detail, builder.ErrHandlerPanic.Error())
}

func TestStartFallsBackToUpdateOnRegisterConflict(t *testing.T) {
	var postCount int
	var putCount int

	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/engine/step" && r.Method == http.MethodPost:
				postCount++
				w.WriteHeader(http.StatusConflict)
			case r.URL.Path == "/engine/step/test-step" &&
				r.Method == http.MethodPut:
				putCount++
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return api.Args{}, nil
	}

	_ = startStepServer(t, engineServer.URL, "test-step", "test-step", handler)

	assert.Equal(t, 1, postCount)
	assert.Equal(t, 1, putCount)
}

func TestHTTPErrorMessage(t *testing.T) {
	err := builder.NewHTTPError(418, "I'm a teapot")
	assert.Equal(t, "HTTP 418: I'm a teapot", err.Error())
}

func TestCompensateHandlerSuccess(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	var gotInput api.Args
	var gotOutput api.Args
	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return api.Args{}, nil
	}
	compensate := func(
		_ *builder.StepContext, input api.Args, output api.Args,
	) error {
		gotInput = input
		gotOutput = output
		return nil
	}

	stepURL := startCompensatingStepServer(t,
		engineServer.URL, "comp-step", "comp-step", handler, compensate,
	)

	body, err := json.Marshal(map[string]api.Args{
		"input":  {"request": "in"},
		"output": {"result": "out"},
	})
	assert.NoError(t, err)

	resp, err := http.Post(
		stepURL+"/compensate", api.JSONContentType, bytes.NewBuffer(body),
	)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, api.Args{"request": "in"}, gotInput)
	assert.Equal(t, api.Args{"result": "out"}, gotOutput)
}

func TestCompensateHandlerRejectsBadRequests(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return api.Args{}, nil
	}
	compensate := func(
		_ *builder.StepContext, _ api.Args, _ api.Args,
	) error {
		return nil
	}

	stepURL := startCompensatingStepServer(t,
		engineServer.URL, "comp-bad", "comp-bad", handler, compensate,
	)

	resp, err := http.Get(stepURL + "/compensate")
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	resp, err = http.Post(
		stepURL+"/compensate", api.JSONContentType,
		bytes.NewBufferString("{"),
	)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCompensateHandlerErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{
			name:   "http-error",
			err:    builder.NewHTTPError(http.StatusConflict, "conflict"),
			status: http.StatusConflict,
		},
		{
			name:   "plain-error",
			err:    assert.AnError,
			status: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engineServer := newHTTPTestServer(t, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/engine/step" &&
						r.Method == http.MethodPost {
						w.WriteHeader(http.StatusCreated)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				},
			))

			handler := func(
				_ *builder.StepContext, _ api.Args,
			) (api.Args, error) {
				return api.Args{}, nil
			}
			compensate := func(
				_ *builder.StepContext, _ api.Args, _ api.Args,
			) error {
				return tc.err
			}

			id := "comp-" + tc.name
			stepURL := startCompensatingStepServer(t,
				engineServer.URL, api.Name(id), api.StepID(id),
				handler, compensate,
			)

			body, err := json.Marshal(map[string]api.Args{
				"input":  {},
				"output": {},
			})
			assert.NoError(t, err)

			resp, err := http.Post(
				stepURL+"/compensate", api.JSONContentType,
				bytes.NewBuffer(body),
			)
			assert.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, tc.status, resp.StatusCode)
		})
	}
}

func TestCompensateHandlerPanic(t *testing.T) {
	engineServer := newHTTPTestServer(t, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	handler := func(
		_ *builder.StepContext, _ api.Args,
	) (api.Args, error) {
		return api.Args{}, nil
	}
	compensate := func(
		_ *builder.StepContext, _ api.Args, _ api.Args,
	) error {
		panic("comp boom")
	}

	stepURL := startCompensatingStepServer(t,
		engineServer.URL, "comp-panic", "comp-panic", handler, compensate,
	)

	body, err := json.Marshal(map[string]api.Args{
		"input":  {},
		"output": {},
	})
	assert.NoError(t, err)

	resp, err := http.Post(
		stepURL+"/compensate", api.JSONContentType, bytes.NewBuffer(body),
	)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var problem api.ProblemDetails
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Contains(t, problem.Detail, builder.ErrHandlerPanic.Error())
}

func getFreePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer func() { _ = l.Close() }()

	_, port, err := net.SplitHostPort(l.Addr().String())
	assert.NoError(t, err)
	return port
}

func startStepServer(
	t *testing.T, engineURL string, stepName api.Name, stepID api.StepID,
	handle builder.StepHandler,
) string {
	t.Helper()

	http.DefaultServeMux = http.NewServeMux()

	port := getFreePort(t)
	host := "127.0.0.1"

	_ = os.Setenv("STEP_PORT", port)
	_ = os.Setenv("STEP_HOSTNAME", host)
	t.Cleanup(func() {
		_ = os.Unsetenv("STEP_PORT")
		_ = os.Unsetenv("STEP_HOSTNAME")
	})

	client := builder.NewClient(engineURL, time.Second)
	go func() {
		_ = client.NewStep().WithName(stepName).
			WithID(string(stepID)).
			WithSyncExecution().
			Start(handle)
	}()

	healthURL := fmt.Sprintf("http://%s:%s/health", host, port)
	assert.Eventually(t, func() bool {
		resp, err := http.Get(healthURL)
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, time.Second, 20*time.Millisecond)

	return fmt.Sprintf("http://%s:%s/%s", host, port, stepID)
}

func startCompensatingStepServer(
	t *testing.T, engineURL string, stepName api.Name, stepID api.StepID,
	handle builder.StepHandler, compensate builder.CompensateHandler,
) string {
	t.Helper()

	http.DefaultServeMux = http.NewServeMux()

	port := getFreePort(t)
	host := "127.0.0.1"

	_ = os.Setenv("STEP_PORT", port)
	_ = os.Setenv("STEP_HOSTNAME", host)
	t.Cleanup(func() {
		_ = os.Unsetenv("STEP_PORT")
		_ = os.Unsetenv("STEP_HOSTNAME")
	})

	client := builder.NewClient(engineURL, time.Second)
	go func() {
		_ = client.NewStep().WithName(stepName).
			WithID(string(stepID)).
			WithSyncExecution().
			WithCompensateHandler(compensate).
			Start(handle)
	}()

	healthURL := fmt.Sprintf("http://%s:%s/health", host, port)
	assert.Eventually(t, func() bool {
		resp, err := http.Get(healthURL)
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, time.Second, 20*time.Millisecond)

	return fmt.Sprintf("http://%s:%s/%s", host, port, stepID)
}
