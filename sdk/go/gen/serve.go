package gen

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	argyll "github.com/kode4food/argyll/sdk/go"
)

// DefaultTimeout is the step execution timeout reported at registration
const DefaultTimeout = 30 * time.Second

// Serve registers the given steps with the engine and serves their handlers.
// The engine URL, hostname and port come from the ARGYLL_ENGINE_URL,
// STEP_HOSTNAME and STEP_PORT environment variables
func Serve(ctx context.Context, steps ...StepDef) error {
	port := envOr("STEP_PORT", strconv.Itoa(argyll.DefaultStepPort))
	host := envOr("STEP_HOSTNAME", "localhost")
	engineURL := envOr("ARGYLL_ENGINE_URL", argyll.DefaultEngineURL)
	base := fmt.Sprintf("http://%s:%s", host, port)

	client := argyll.NewClient(engineURL, DefaultTimeout)
	if err := Register(ctx, client, base, steps...); err != nil {
		return err
	}

	slog.Info("Step server starting", slog.String("endpoint", base))
	server := &http.Server{Addr: ":" + port, Handler: Mux(steps...)}
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
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", api.JSONContentType)
		_, _ = fmt.Fprint(w, `{"status": "healthy"}`)
	})
	for _, s := range steps {
		mux.HandleFunc("/"+string(s.ID), s.Handler)
	}
	return mux
}

func registerStep(
	ctx context.Context, client *argyll.Client, base string, s StepDef,
) error {
	step := client.NewStep().WithName(s.Name).WithID(string(s.ID)).
		WithType(s.Type).WithLabels(s.Labels).
		WithEndpoint(base + "/" + string(s.ID)).
		WithHealthCheck(base + "/health")

	for _, a := range s.Inputs {
		if a.Optional {
			step = step.Optional(a.Name, a.Type, "")
			continue
		}
		step = step.Required(a.Name, a.Type)
	}
	for _, a := range s.Outputs {
		step = step.Output(a.Name, a.Type)
	}
	return step.Register(ctx)
}

func envOr(name, defaultValue string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultValue
}
