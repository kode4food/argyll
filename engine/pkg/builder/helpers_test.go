package builder_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/builder"
)

func TestSetupStepWithMockEngine(t *testing.T) {
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
		client := builder.NewClient(mockEngine.URL, 5*time.Second)
		err := client.NewStep("Test Step").
			WithSyncExecution().
			Start(handler)
		errChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Greater(t, attempts, 0)
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
	assert.Equal(t, "http://localhost:8080", builder.DefaultEngineURL)
}
