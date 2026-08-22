package argyll

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

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// StepAddr is the port a step server listens on and the base URL it
	// advertises to the engine
	StepAddr struct {
		BaseURL string
		Port    string
	}

	compensateBody struct {
		Input  api.Args `json:"input"`
		Output api.Args `json:"output"`
	}
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

// LocalStepAddr reads the step server address from the STEP_HOSTNAME and
// STEP_PORT environment variables
func LocalStepAddr() StepAddr {
	port := EnvOr("STEP_PORT", strconv.Itoa(DefaultStepPort))
	host := EnvOr("STEP_HOSTNAME", "localhost")
	return StepAddr{
		BaseURL: fmt.Sprintf("http://%s:%s", host, port),
		Port:    port,
	}
}

// EnvOr returns the named environment variable, or the given default when it is
// unset or empty
func EnvOr(name, defaultValue string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultValue
}

// HealthHandler serves the health endpoint the engine polls. An empty service
// name is reported as a bare status
func HealthHandler(service api.Name) http.HandlerFunc {
	body := `{"status": "healthy"}`
	if service != "" {
		body = fmt.Sprintf(`{"status": "healthy", "service": "%s"}`, service)
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}
}

// WriteProblem writes an RFC 7807 problem response
func WriteProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", api.ProblemJSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.NewProblem(status, detail))
}

func setupStepServer(client *Client, step Step, handle StepHandler) error {
	addr := LocalStepAddr()
	id := step.step.ID
	endpoint := fmt.Sprintf("%s/%s", addr.BaseURL, id)
	step = step.WithEndpoint(endpoint).
		WithHealthCheck(addr.BaseURL + "/health")

	if step.compensate != nil && step.step.HTTP.Compensate == nil {
		step = step.WithCompensate(endpoint + "/compensate")
	}

	stepReq, err := step.Build()
	if err != nil {
		return err
	}
	if err := client.RegisterStep(context.Background(), stepReq); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler(api.Name(id)))

	handler := makeStepHandler(client, id, handle)
	mux.HandleFunc("/"+string(id), handler)

	if step.compensate != nil {
		compHandler := makeCompensateHandler(client, id, step.compensate)
		mux.HandleFunc("/"+string(id)+"/compensate", compHandler)
	}

	slog.Info("Step server starting",
		slog.String("step_name", string(step.step.Name)),
		log.StepID(id),
		slog.String("port", addr.Port),
		slog.String("endpoint", endpoint))
	server := &http.Server{
		Addr:    ":" + addr.Port,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func makeCompensateHandler(
	client *Client, id api.StepID, handler CompensateHandler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteProblem(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var body compensateBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteProblem(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		meta := api.MetadataFromHeaders(r.Header)
		fid, _ := meta.GetString[api.FlowID](api.MetaFlowID)

		ctx := &StepContext{
			Context:  r.Context(),
			Client:   client.Flow(fid),
			StepID:   id,
			Metadata: meta,
		}

		httpErr := executeCompensateWithRecovery(ctx, handler, body)
		if httpErr != nil {
			WriteProblem(w, httpErr.StatusCode, httpErr.Message)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func makeStepHandler(
	client *Client, id api.StepID, handler StepHandler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteProblem(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var args api.Args
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			WriteProblem(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		meta := api.MetadataFromHeaders(r.Header)
		fid, _ := meta.GetString[api.FlowID](api.MetaFlowID)

		ctx := &StepContext{
			Context:  r.Context(),
			Client:   client.Flow(fid),
			StepID:   id,
			Metadata: meta,
		}
		outputs, err := executeStepWithRecovery(ctx, handler, args)
		if err != nil {
			WriteProblem(w, err.StatusCode, err.Message)
			return
		}

		w.Header().Set("Content-Type", api.JSONContentType)
		_ = json.NewEncoder(w).Encode(outputs)
	}
}

func executeStepWithRecovery(
	ctx *StepContext, handler StepHandler, args api.Args,
) (outputs api.Args, httpErr *HTTPError) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Step handler panicked",
				log.StepID(ctx.StepID),
				log.Error(ErrHandlerPanic),
				slog.String("panic", fmt.Sprintf("%v", r)))
			httpErr = NewHTTPError(
				http.StatusInternalServerError,
				fmt.Sprintf("%s: %v", ErrHandlerPanic, r),
			)
		}
	}()

	var err error
	outputs, err = handler(ctx, args)
	if err != nil {
		if he, ok := errors.AsType[*HTTPError](err); ok {
			return nil, he
		}
		return nil, NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return outputs, nil
}

func executeCompensateWithRecovery(
	ctx *StepContext, handler CompensateHandler, body compensateBody,
) (httpErr *HTTPError) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Compensate handler panicked",
				log.StepID(ctx.StepID),
				log.Error(ErrHandlerPanic),
				slog.String("panic", fmt.Sprintf("%v", r)))
			httpErr = NewHTTPError(
				http.StatusInternalServerError,
				fmt.Sprintf("%s: %v", ErrHandlerPanic, r),
			)
		}
	}()
	if err := handler(ctx, body.Input, body.Output); err != nil {
		if he, ok := errors.AsType[*HTTPError](err); ok {
			return he
		}
		return NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}
