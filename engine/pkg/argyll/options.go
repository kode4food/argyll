package argyll

import (
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/config"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type (
	// Options configures an embedded Engine. A nil Config uses the defaults.
	// A nil Handlers map installs all standard handlers; a non-nil map is used
	// exactly as supplied
	Options struct {
		Backend  OpenBackend
		Config   *config.Config
		Handlers step.Handlers
	}

	// OpenBackend opens the timebox backend an Engine runs on, wired to the
	// publisher its committed events must reach
	OpenBackend func(timebox.Publisher) (timebox.Backend, error)
)

var _ Engine = (*engine.Engine)(nil)

// New creates an Engine over the backend Options opens. Nothing is processed
// until Start is called
func New(opts Options) (Engine, error) {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}

	eng, err := engine.New(cfg, engine.Dependencies{
		Scripts:  script.NewRegistry(),
		Steps:    step.NewRegistry(opts.handlers(cfg)),
		EventHub: event.NewHub(),
	}, engine.OpenBackend(opts.Backend))
	if err != nil {
		return nil, err
	}
	return eng, nil
}

// handlers returns the exact caller-supplied set. A nil set preserves the
// default configuration for callers that do not customize handlers
func (o Options) handlers(cfg *config.Config) step.Handlers {
	if o.Handlers != nil {
		return o.Handlers
	}
	return builtins.All(
		builtins.NewHTTPClient(
			time.Duration(cfg.StepTimeout)*time.Millisecond,
		),
		builtins.BaseCallbackURL(cfg.WebhookBaseURL),
	)
}
