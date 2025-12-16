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
	"github.com/kode4food/argyll/engine/pkg/builder"
)

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
	t *testing.T,
	engineURL string,
	stepName api.Name,
	stepID api.StepID,
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
		_ = client.NewStep(stepName).
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

func TestHandlerHTTPError(t *testing.T) {
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
	) (api.StepResult, error) {
		return api.StepResult{},
			builder.NewHTTPError(http.StatusTeapot, "teapot")
	}

	stepURL := startStepServer(
		t, engineServer.URL, "test-step", "test-step", handler,
	)

	req := api.StepRequest{
		Arguments: api.Args{"foo": "bar"},
		Metadata:  api.Metadata{"flow_id": "flow-1"},
	}
	body, err := json.Marshal(req)
	assert.NoError(t, err)

	resp, err := http.Post(stepURL, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
}

func TestHandlerPanic(t *testing.T) {
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
	) (api.StepResult, error) {
		panic("boom")
	}

	stepURL := startStepServer(
		t, engineServer.URL, "panic-step", "panic-step", handler,
	)

	req := api.StepRequest{
		Arguments: api.Args{},
		Metadata:  api.Metadata{"flow_id": "flow-2"},
	}
	body, err := json.Marshal(req)
	assert.NoError(t, err)

	resp, err := http.Post(stepURL, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result api.StepResult
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, builder.ErrHandlerPanic.Error())
}
