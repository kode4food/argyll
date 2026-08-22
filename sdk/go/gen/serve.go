package gen

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	argyll "github.com/kode4food/argyll/sdk/go"
)

// DefaultTimeout is the engine client timeout used while registering
const DefaultTimeout = 30 * time.Second

// Serve registers the given steps with the engine and serves their handlers.
// The engine URL, hostname and port come from the ARGYLL_ENGINE_URL,
// STEP_HOSTNAME and STEP_PORT environment variables
func Serve(ctx context.Context, steps ...StepDef) error {
	addr := argyll.LocalStepAddr()
	engineURL := argyll.EnvOr("ARGYLL_ENGINE_URL", argyll.DefaultEngineURL)

	client := argyll.NewClient(engineURL, DefaultTimeout)
	if err := Register(ctx, client, addr.BaseURL, steps...); err != nil {
		return err
	}

	slog.Info("Step server starting", slog.String("endpoint", addr.BaseURL))
	server := &http.Server{Addr: ":" + addr.Port, Handler: Mux(steps...)}
	return server.ListenAndServe()
}

// Register registers the given steps with the engine, pointing them at the
// handlers served under the base URL
func Register(
	ctx context.Context, client *argyll.Client, base string,
	steps ...StepDef,
) error {
	for _, s := range steps {
		if err := registerStep(ctx, client, base, s); err != nil {
			return err
		}
	}
	return nil
}

// Mux returns a handler serving the steps and a health endpoint
func Mux(steps ...StepDef) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", argyll.HealthHandler(""))
	for _, s := range steps {
		mux.HandleFunc("/"+string(s.ID), s.Handler)
	}
	return mux
}

// the generator writes endpoints as paths, this adds the serving host
func registerStep(
	ctx context.Context, client *argyll.Client, base string, s StepDef,
) error {
	step, err := s.Step()
	if err != nil {
		return err
	}
	step.HTTP.Invoke.Endpoint = base + step.HTTP.Invoke.Endpoint
	step.HTTP.Health = base + step.HTTP.Health
	if step.HTTP.Compensate != nil {
		step.HTTP.Compensate.Endpoint = base + step.HTTP.Compensate.Endpoint
	}
	return client.RegisterStep(ctx, step)
}
