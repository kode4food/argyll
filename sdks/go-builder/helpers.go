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

func setupStepServer(client *Client, step Step, handle StepHandler) error {
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
		return fmt.Errorf("%w: %d attempts", ErrStepRegistration,
			MaxRegistrationAttempts)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
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
			writeProblem(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var args api.Args
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		meta := api.MetadataFromHeaders(r.Header)
		fid, _ := api.GetMetaString[api.FlowID](meta, api.MetaFlowID)

		ctx := &StepContext{
			Context:  r.Context(),
			Client:   client.Flow(fid),
			StepID:   id,
			Metadata: meta,
		}
		outputs, err := executeStepWithRecovery(ctx, id, handler, args)
		if err != nil {
			writeProblem(w, err.StatusCode, err.Message)
			return
		}

		w.Header().Set("Content-Type", api.JSONContentType)
		_ = json.NewEncoder(w).Encode(outputs)
	}
}

func executeStepWithRecovery(
	ctx *StepContext, id api.StepID, handler StepHandler, args api.Args,
) (outputs api.Args, httpErr *HTTPError) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Step handler panicked",
				log.StepID(id),
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
		var he *HTTPError
		if errors.As(err, &he) {
			return nil, he
		}
		return nil, NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return outputs, nil
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", api.ProblemJSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.NewProblem(status, detail))
}
