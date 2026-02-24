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
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

const (
	MaxRegistrationAttempts = 5
	BackoffMultiplier       = 2 * time.Second
	DefaultEngineURL        = "http://localhost:8080"
)

var (
	ErrStepRegistration = errors.New("failed to register step after retries")
	ErrHandlerPanic     = errors.New("step handler panicked")
)

func setupStepServer(client *Client, step *Step, handle StepHandler) error {
	port := os.Getenv("STEP_PORT")
	if port == "" {
		port = strconv.Itoa(DefaultStepPort)
	}

	portInt, _ := strconv.Atoi(port)

	hostname := os.Getenv("STEP_HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}

	endpoint := fmt.Sprintf("http://%s:%d/%s", hostname, portInt, step.id)
	healthEndpoint := fmt.Sprintf("http://%s:%d/health", hostname, portInt)

	step = step.WithEndpoint(endpoint).WithHealthCheck(healthEndpoint)

	stepReq, err := step.Build()
	if err != nil {
		return err
	}

	var registered bool
	for attempt := 1; attempt <= MaxRegistrationAttempts; attempt++ {
		var err error
		if step.dirty {
			err = client.updateStep(context.Background(), stepReq)
		} else {
			err = client.registerStep(context.Background(), stepReq)
			if err != nil && isRegisterConflict(err) {
				err = client.updateStep(context.Background(), stepReq)
			}
		}

		if err == nil {
			registered = true
			break
		}

		if attempt >= MaxRegistrationAttempts {
			continue
		}
		backoff := time.Duration(attempt) * BackoffMultiplier
		time.Sleep(backoff)
	}

	if !registered {
		return fmt.Errorf("%w: %d attempts",
			ErrStepRegistration, MaxRegistrationAttempts)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status": "healthy", "service": "%s"}`,
			string(step.id))
	})

	handler := makeStepHandler(client, step.id, handle)
	mux.HandleFunc("/"+string(step.id), handler)

	slog.Info("Step server starting",
		slog.String("step_name", string(step.name)),
		log.StepID(step.id),
		slog.String("port", port),
		slog.String("endpoint", endpoint))
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func isRegisterConflict(err error) bool {
	return errors.Is(err, ErrRegisterStep) &&
		strings.Contains(err.Error(), "status 409")
}

func makeStepHandler(
	client *Client, id api.StepID, handler StepHandler,
) http.HandlerFunc {
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

		var flowID api.FlowID
		if req.Metadata != nil {
			if fid, ok := req.Metadata[api.MetaFlowID].(string); ok {
				flowID = api.FlowID(fid)
			}
		}

		ctx := &StepContext{
			Context:  r.Context(),
			Client:   client.Flow(flowID),
			StepID:   id,
			Metadata: req.Metadata,
		}
		result, err := executeStepWithRecovery(ctx, id, handler, req.Arguments)
		if err != nil {
			http.Error(w, err.Message, err.StatusCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func executeStepWithRecovery(
	ctx *StepContext, id api.StepID, handler StepHandler, args api.Args,
) (result api.StepResult, httpErr *HTTPError) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Step handler panicked",
				log.StepID(id),
				log.Error(ErrHandlerPanic),
				slog.String("panic", fmt.Sprintf("%v", r)))
			result = *api.NewResult().WithError(
				fmt.Errorf("%w: %v", ErrHandlerPanic, r),
			)
			httpErr = nil
		}
	}()

	var err error
	result, err = handler(ctx, args)
	if err != nil {
		var he *HTTPError
		if errors.As(err, &he) {
			return api.StepResult{}, he
		}
		return *api.NewResult().WithError(err), nil
	}
	return result, nil
}
