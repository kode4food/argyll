package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type ContextKey string

const (
	MaxRegistrationAttempts = 5
	BackoffMultiplier       = 2 * time.Second
	DefaultEngineURL        = "http://localhost:8080"
)

var (
	ErrStepRegistration = errors.New("failed to register step after retries")
	ErrHandlerPanic     = errors.New("step handler panicked")
)

var MetadataKey ContextKey = "metadata"

func SetupStep(
	name api.Name, build func(*Step) *Step, handle api.StepHandler,
) error {
	step := build(NewStep(name))
	id := step.id

	port := os.Getenv("STEP_PORT")
	if port == "" {
		port = strconv.Itoa(DefaultStepPort)
	}

	engineURL := os.Getenv("SPUDS_ENGINE_URL")
	if engineURL == "" {
		engineURL = DefaultEngineURL
	}

	client := NewClient(engineURL, 30*time.Second)

	portInt, _ := strconv.Atoi(port)

	hostname := os.Getenv("STEP_HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}

	endpoint := fmt.Sprintf("http://%s:%d/%s", hostname, portInt, id)
	healthEndpoint := fmt.Sprintf("http://%s:%d/health", hostname, portInt)

	step = step.WithEndpoint(endpoint).WithHealthCheck(healthEndpoint)
	stepReq, err := step.Build()
	if err != nil {
		return err
	}

	var registered bool
	for attempt := 1; attempt <= MaxRegistrationAttempts; attempt++ {
		err := client.RegisterStep(context.Background(), stepReq)
		if err == nil {
			registered = true
			break
		}

		slog.Warn("Failed to register step",
			slog.Any("step_id", id),
			slog.Int("attempt", attempt),
			slog.Any("error", err))
		if attempt >= MaxRegistrationAttempts {
			continue
		}
		backoff := time.Duration(attempt) * BackoffMultiplier
		time.Sleep(backoff)
	}

	if !registered {
		return fmt.Errorf("%s: %d attempts",
			ErrStepRegistration, MaxRegistrationAttempts)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status": "healthy", "service": "%s"}`, id)
	})

	http.HandleFunc("/"+string(id), makeStepHandler(id, handle))

	slog.Info("Step server starting",
		slog.Any("step_name", name),
		slog.Any("step_id", id),
		slog.String("port", port),
		slog.String("endpoint", endpoint))
	return http.ListenAndServe(":"+port, nil)
}

func makeStepHandler(id timebox.ID, handler api.StepHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req api.StepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), MetadataKey, req.Metadata)
		result := executeStepWithRecovery(ctx, id, handler, req.Arguments)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func executeStepWithRecovery(
	ctx context.Context, id timebox.ID, handler api.StepHandler, args api.Args,
) (result api.StepResult) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Step handler panicked",
				slog.Any("step_id", id),
				slog.Any("panic", r))
			result = *api.NewResult().WithError(
				fmt.Errorf("%s: %v", ErrHandlerPanic, r),
			)
		}
	}()

	var err error
	result, err = handler(ctx, args)
	if err != nil {
		return *api.NewResult().WithError(err)
	}
	return result
}
