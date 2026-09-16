package argyll

import (
	"context"
	"errors"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type (
	// Options configures an embedded Engine. A nil Handlers map installs all
	// standard handlers; a non-nil map is used exactly as supplied
	Options struct {
		Raft     raft.Config
		Handlers step.Handlers
	}

	// embedded is the engine plus the stores New opened on its behalf, which
	// is all that separates an embedded engine from the engine itself
	embedded struct {
		*engine.Engine
		backend *raft.Backend
	}
)

const DefaultStoreReadyTimeout = 5 * time.Second

var (
	ErrCreateStore = errors.New("failed to create raft store")
)

var _ Engine = (*embedded)(nil)

// New opens the stores an embedded Engine runs on. Nothing is processed until
// Start is called
func New(opts Options) (Engine, error) {
	cfg := opts.config()
	hub := event.NewHub()
	res := &embedded{}
	cfg.Raft.Publisher = func(evs ...*timebox.Event) {
		if res.Engine != nil {
			res.Engine.HandleCommitted(evs...)
		}
		hub.Publish(evs...)
	}

	b, err := raft.Open(cfg.Raft)
	if err != nil {
		return nil, errors.Join(ErrCreateStore, err)
	}
	engStore, err := b.NewStore(cfg.EngineStoreConfig())
	if err != nil {
		return nil, closeOnError(b, err)
	}
	flowStore, err := b.NewStore(cfg.FlowStoreConfig())
	if err != nil {
		return nil, closeOnError(b, err)
	}
	ctx, cancel := context.WithTimeout(
		context.Background(), DefaultStoreReadyTimeout,
	)
	defer cancel()
	if err := flowStore.WaitReady(ctx); err != nil {
		return nil, closeOnError(b, err)
	}

	scripts := script.NewRegistry()
	eng, err := engine.New(cfg, engine.Dependencies{
		EngineStore: engStore,
		FlowStore:   flowStore,
		Scripts:     scripts,
		Steps:       step.NewRegistry(opts.handlers(cfg)),
		EventHub:    hub,
	})
	if err != nil {
		return nil, closeOnError(b, err)
	}

	res.Engine = eng
	res.backend = b
	return res, nil
}

// Stop shuts the engine down and closes the stores New opened
func (e *embedded) Stop() error {
	return errors.Join(e.Engine.Stop(), e.backend.Close())
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

func (o Options) config() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.Raft = cfg.Raft.With(o.Raft)
	return cfg
}

func closeOnError(b *raft.Backend, err error) error {
	_ = b.Close()
	return errors.Join(ErrCreateStore, err)
}
