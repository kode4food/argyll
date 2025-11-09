package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestWithRecoverySuccess(t *testing.T) {
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{
			Success: true,
			Outputs: api.Args{"result": "success"},
		}, nil
	}

	result := executeStepWithRecovery(
		context.Background(), "test-step", handler, api.Args{},
	)

	assert.True(t, result.Success)
	assert.Equal(t, "success", result.Outputs["result"])
	assert.Empty(t, result.Error)
}

func TestWithRecoveryError(t *testing.T) {
	expectedErr := errors.New("handler error")
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{}, expectedErr
	}

	result := executeStepWithRecovery(
		context.Background(), "test-step", handler, api.Args{},
	)

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "handler error")
}

func TestWithRecoveryPanic(t *testing.T) {
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		panic("something went wrong")
	}

	result := executeStepWithRecovery(
		context.Background(), "test-step", handler, api.Args{},
	)

	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "step handler panicked")
	assert.Contains(t, result.Error, "something went wrong")
}

func TestSuccess(t *testing.T) {
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{
			Success: true,
			Outputs: api.Args{"result": args["input"]},
		}, nil
	}

	stepHandler := makeStepHandler("test-step", handler)

	reqBody := api.StepRequest{
		Arguments: api.Args{"input": "test-value"},
		Metadata:  api.Metadata{"flow_id": "wf-123"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost, "/test-step", bytes.NewReader(body),
	)
	w := httptest.NewRecorder()

	stepHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result api.StepResult
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "test-value", result.Outputs["result"])
}

func TestMethodNotAllowed(t *testing.T) {
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{Success: true}, nil
	}

	stepHandler := makeStepHandler("test-step", handler)

	req := httptest.NewRequest(http.MethodGet, "/test-step", nil)
	w := httptest.NewRecorder()

	stepHandler(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestInvalidJSON(t *testing.T) {
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{Success: true}, nil
	}

	stepHandler := makeStepHandler("test-step", handler)

	req := httptest.NewRequest(
		http.MethodPost, "/test-step", bytes.NewReader([]byte("invalid json")),
	)
	w := httptest.NewRecorder()

	stepHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWithMetadata(t *testing.T) {
	var capturedCtx context.Context
	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		capturedCtx = ctx
		return api.StepResult{Success: true}, nil
	}

	stepHandler := makeStepHandler("test-step", handler)

	reqBody := api.StepRequest{
		Arguments: api.Args{},
		Metadata:  api.Metadata{"flow_id": "wf-123", "user_id": "user-456"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost, "/test-step", bytes.NewReader(body),
	)
	w := httptest.NewRecorder()

	stepHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedCtx)

	metadata := capturedCtx.Value(MetadataKey)
	require.NotNil(t, metadata)
	meta, ok := metadata.(api.Metadata)
	require.True(t, ok)
	assert.Equal(t, "wf-123", meta["flow_id"])
	assert.Equal(t, "user-456", meta["user_id"])
}

func TestSetupEnvironmentVariables(t *testing.T) {
	_ = os.Setenv("STEP_PORT", "9876")
	_ = os.Setenv("SPUDS_ENGINE_URL", "http://test-engine:8080")
	_ = os.Setenv("STEP_HOSTNAME", "test-host")
	unset := func() {
		_ = os.Unsetenv("STEP_PORT")
		_ = os.Unsetenv("SPUDS_ENGINE_URL")
		_ = os.Unsetenv("STEP_HOSTNAME")
	}
	defer unset()
	unset()
	assert.Equal(t, "http://localhost:8080", DefaultEngineURL)
}

func TestSetupWithMockEngine(t *testing.T) {
	attempts := 0
	mockEngine := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if r.URL.Path == "/engine/step" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))
	defer mockEngine.Close()

	_ = os.Setenv("SPUDS_ENGINE_URL", mockEngine.URL)
	_ = os.Setenv("STEP_PORT", "0")
	defer func() {
		_ = os.Unsetenv("SPUDS_ENGINE_URL")
		_ = os.Unsetenv("STEP_PORT")
	}()

	handler := func(
		ctx context.Context, args api.Args,
	) (api.StepResult, error) {
		return api.StepResult{Success: true}, nil
	}

	errChan := make(chan error, 1)
	go func() {
		err := SetupStep(
			"Test Step",
			func(b *Step) *Step {
				return b.WithSyncExecution()
			},
			handler,
		)
		errChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Greater(t, attempts, 0, "Engine should have received registration")
}
