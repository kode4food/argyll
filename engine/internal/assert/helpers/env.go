package helpers

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type (
	// TestEngineEnv holds all the components needed for engine testing
	TestEngineEnv struct {
		T          *testing.T
		Engine     *engine.Engine
		MockClient *MockClient
		Config     *config.Config
		EventHub   *event.Hub
		Cleanup    func()
		backend    *backend
		flowStore  *timebox.Store
		flowExec   *timebox.Executor[api.FlowState]
		subscribe  func(timebox.Publisher) func()
		unsubs     *unsubscribeTracker
		conflict   *conflictOnce
		ownsHub    bool
	}

	FlowEvent struct {
		Data any
		Type api.EventType
	}

	// backend is the backend every engine in a test shares. Each engine opens
	// it with its own publisher, and all of them hear every commit
	backend struct {
		timebox.Backend
		publish  timebox.Publisher
		conflict *conflictOnce
	}

	// sharedBackend is one engine's view of the shared backend. The test owns
	// the real backend, so an engine stopping cannot close it for the others
	sharedBackend struct {
		*backend
	}

	// conflictOnce makes one append to a chosen aggregate report a stale
	// sequence, which is what the executor sees when another writer wins
	conflictOnce struct {
		target timebox.AggregateID
		armed  bool
		fired  bool
		mu     sync.Mutex
	}

	unsubscribeTracker struct {
		funcs []func()
		mu    sync.Mutex
	}
)

var ErrInvalidSeedFlowStatus = errors.New("invalid seeded flow status")

// WithTestEnv creates a test engine environment, executes the provided
// function with it, and ensures cleanup happens automatically
func WithTestEnv(t *testing.T, fn func(*TestEngineEnv)) {
	WithTestEnvDeps(t, engine.Dependencies{}, fn)
}

// WithTestEnvDeps creates a test engine environment with dependency
// overrides and ensures cleanup happens automatically
func WithTestEnvDeps(
	t *testing.T, overrides engine.Dependencies, fn func(*TestEngineEnv),
) {
	t.Helper()
	testEnv := NewTestEngineWithDeps(t, overrides)
	defer testEnv.Cleanup()
	fn(testEnv)
}

// WithTestBackend creates a test engine environment whose engines share the
// provided backend, so a test can inject faults into its writes
func WithTestBackend(
	t *testing.T, b timebox.Backend, fn func(*TestEngineEnv),
) {
	t.Helper()
	testEnv := newTestEngine(t, engine.Dependencies{}, b)
	defer testEnv.Cleanup()
	fn(testEnv)
}

// WithEngine creates a test engine, executes the provided function with it,
// and ensures cleanup happens automatically
func WithEngine(t *testing.T, fn func(*engine.Engine)) {
	WithEngineDeps(t, engine.Dependencies{}, fn)
}

// WithEngineDeps creates a test engine with dependency overrides and
// ensures cleanup happens automatically
func WithEngineDeps(
	t *testing.T, overrides engine.Dependencies, fn func(*engine.Engine),
) {
	t.Helper()
	WithTestEnvDeps(t, overrides, func(env *TestEngineEnv) {
		fn(env.Engine)
	})
}

// WithStartedEngine creates a test engine, starts it, executes the provided
// function with the engine, and ensures cleanup happens automatically
func WithStartedEngine(t *testing.T, fn func(*engine.Engine)) {
	t.Helper()
	WithEngine(t, func(eng *engine.Engine) {
		assert.NoError(t, eng.Start())
		fn(eng)
	})
}

// NewTestEngine creates a fully configured test engine environment with an
// in-memory Timebox backend and mock HTTP client
func NewTestEngine(t *testing.T) *TestEngineEnv {
	return NewTestEngineWithDeps(t, engine.Dependencies{})
}

// NewTestEngineWithDeps creates a test engine with dependency overrides
func NewTestEngineWithDeps(
	t *testing.T, overrides engine.Dependencies,
) *TestEngineEnv {
	t.Helper()
	return newTestEngine(t, overrides, memory.Open())
}

// OpenBackend hands an engine the backend every engine in this test shares,
// subscribing it to the commits all of them make
func (e *TestEngineEnv) OpenBackend(
	pub timebox.Publisher,
) (timebox.Backend, error) {
	e.trackUnsubscribe(e.SubscribeCommitted(pub))
	return sharedBackend{backend: e.backend}, nil
}

// Close leaves the shared backend open, since the test owns it
func (sharedBackend) Close() error {
	return nil
}

// SubscribeCommitted registers a publisher against the shared committed-event
// stream used by test engines. Call the returned function to unregister it
func (e *TestEngineEnv) SubscribeCommitted(fn timebox.Publisher) func() {
	return e.subscribe(fn)
}

// NewEngineWithConfig creates a new engine instance with the given
// configuration and subscribes it to the shared committed-event stream
func (e *TestEngineEnv) NewEngineWithConfig(
	cfg *config.Config, deps engine.Dependencies,
) (*engine.Engine, func(), error) {
	var unsubscribe func()
	eng, err := engine.New(cfg, deps,
		func(pub timebox.Publisher) (timebox.Backend, error) {
			unsubscribe = e.SubscribeCommitted(pub)
			e.trackUnsubscribe(unsubscribe)
			return sharedBackend{backend: e.backend}, nil
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return eng, unsubscribe, nil
}

// NewEngineInstance creates a new engine instance sharing the same backend
// and mock client. Used to simulate process restart after crash
func (e *TestEngineEnv) NewEngineInstance() (*engine.Engine, error) {
	eng, err := engine.New(e.Config, e.Dependencies(), e.OpenBackend)
	if err != nil {
		return nil, err
	}
	e.flowExec = e.flowStore.Executor(
		events.NewFlowState, events.FlowAppliers,
	)
	return eng, nil
}

// Dependencies returns a valid dependency bundle for constructing an engine
func (e *TestEngineEnv) Dependencies() engine.Dependencies {
	return e.engineDeps(time.Now, scheduler.NewTimer)
}

// RaiseFlowEvents appends flow events via the executor
func (e *TestEngineEnv) RaiseFlowEvents(
	fid api.FlowID, evs ...FlowEvent,
) error {
	_, err := e.flowExec.Exec(
		events.FlowKey(fid),
		func(_ api.FlowState, ag *timebox.Aggregator[api.FlowState]) error {
			for _, ev := range evs {
				if err := raiseFlowEvent(ag, ev); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

// SeedFlow stores a minimal valid flow history for state/query fixtures
func (e *TestEngineEnv) SeedFlow(
	fid api.FlowID, status api.FlowStatus, tags api.Tags,
) error {
	evs := []FlowEvent{{
		Type: api.EventTypeFlowStarted,
		Data: api.FlowStartedEvent{
			FlowID: fid,
			Plan:   &api.ExecutionPlan{Steps: api.Steps{}},
			Tags:   tags,
		},
	}}
	switch status {
	case api.FlowActive:
		return e.RaiseFlowEvents(fid, evs...)
	case api.FlowCompleted:
		evs = append(evs, FlowEvent{
			Type: api.EventTypeFlowCompleted,
			Data: api.FlowCompletedEvent{FlowID: fid},
		})
	case api.FlowFailed:
		evs = append(evs, FlowEvent{
			Type: api.EventTypeFlowFailed,
			Data: api.FlowFailedEvent{FlowID: fid},
		})
	default:
		return ErrInvalidSeedFlowStatus
	}
	evs = append(evs, FlowEvent{
		Type: api.EventTypeFlowDeactivated,
		Data: api.FlowDeactivatedEvent{FlowID: fid, Status: status},
	})
	return e.RaiseFlowEvents(fid, evs...)
}

// SeedStartedWork stores a flow whose single work item has been claimed, the
// state a node leaves behind when it dies mid-attempt
func (e *TestEngineEnv) SeedStartedWork(
	fs api.FlowStep, pl *api.ExecutionPlan, tkn api.Token,
) error {
	return e.RaiseFlowEvents(fs.FlowID,
		FlowEvent{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: fs.FlowID,
				Plan:   pl,
				Init:   api.InitArgs{},
			},
		},
		FlowEvent{
			Type: api.EventTypeStepStarted,
			Data: api.StepStartedEvent{
				FlowID:    fs.FlowID,
				StepID:    fs.StepID,
				Inputs:    api.Args{},
				WorkItems: map[api.Token]api.Args{tkn: {}},
			},
		},
		FlowEvent{
			Type: api.EventTypeWorkStarted,
			Data: api.WorkStartedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  tkn,
				Inputs: api.Args{},
			},
		},
	)
}

// ConflictOnNextAppend makes the next append for a flow lose an optimistic-
// concurrency race, so the executor discards that attempt and runs the command
// again. Pair it with ConflictFired to confirm it took effect
func (e *TestEngineEnv) ConflictOnNextAppend(fid api.FlowID) {
	e.conflict.arm(events.FlowKey(fid))
}

// ConflictFired reports whether an armed conflict was actually injected
func (e *TestEngineEnv) ConflictFired() bool {
	return e.conflict.hasFired()
}

// AppendEvents appends raw events to the shared test store
func (e *TestEngineEnv) AppendEvents(
	id timebox.AggregateID, atSeq int64, evs ...*timebox.Event,
) error {
	return e.flowStore.AppendEvents(id, atSeq, evs)
}

// ListFlowsByTag returns the flow aggregate IDs currently indexed for the tag
func (e *TestEngineEnv) ListFlowsByTag(
	tag string,
) ([]timebox.AggregateID, error) {
	return e.flowStore.ListAggregatesByTag(tag)
}

func (e *TestEngineEnv) trackUnsubscribe(fn func()) {
	e.unsubs.mu.Lock()
	defer e.unsubs.mu.Unlock()
	e.unsubs.funcs = append(e.unsubs.funcs, fn)
}

func (e *TestEngineEnv) unsubscribeAll() {
	e.unsubs.mu.Lock()
	unsubs := e.unsubs.funcs
	e.unsubs.funcs = nil
	e.unsubs.mu.Unlock()

	for _, fn := range unsubs {
		fn()
	}
}

func (e *TestEngineEnv) engineDeps(
	clock scheduler.Clock, makeTimer scheduler.TimerConstructor,
) engine.Dependencies {
	scripts := script.NewRegistry()
	steps := step.NewRegistry(builtins.All(
		e.MockClient,
		builtins.BaseCallbackURL(e.Config.WebhookBaseURL),
	))

	// Every engine publishes what it hears to its own hub, so a second engine
	// on the test's hub would deliver each event twice
	return engine.Dependencies{
		Scripts:          scripts,
		Steps:            steps,
		Clock:            clock,
		TimerConstructor: makeTimer,
		EventHub:         event.NewHub(),
	}
}

func (b backend) Append(reqs ...timebox.AppendRequest) error {
	// an armed conflict fails the whole set, as a real one would
	for _, req := range reqs {
		if b.conflict.take(req.ID) {
			return &timebox.VersionConflictError{
				ID:               req.ID,
				ExpectedSequence: req.ExpectedSequence,
				ActualSequence:   req.ExpectedSequence + 1,
			}
		}
	}
	if err := b.Backend.Append(reqs...); err != nil {
		return err
	}
	for _, req := range reqs {
		if len(req.Events) != 0 {
			b.publish(req.Events...)
		}
	}
	return nil
}

func (c *conflictOnce) arm(id timebox.AggregateID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = id
	c.armed = true
	c.fired = false
}

func (c *conflictOnce) take(id timebox.AggregateID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.armed || id != c.target {
		return false
	}
	c.armed = false
	c.fired = true
	return true
}

func (c *conflictOnce) hasFired() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fired
}

func newTestEngine(
	t *testing.T, overrides engine.Dependencies, b timebox.Backend,
) *TestEngineEnv {
	t.Helper()

	cfg := config.NewDefaultConfig()
	cfg.StepTimeout = 5 * api.Second
	cfg.MemoCacheSize = 100
	cfg.Work = api.WorkConfig{
		MaxRetries:  3,
		InitBackoff: 1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	mockCli := NewMockClient()
	hub := overrides.EventHub
	ownsHub := false
	if hub == nil {
		hub = event.NewHub()
		ownsHub = true
	}
	var publishMu sync.Mutex
	committed := map[int]timebox.Publisher{}
	nextCommittedID := 0
	subscribe := func(fn timebox.Publisher) func() {
		publishMu.Lock()
		id := nextCommittedID
		nextCommittedID++
		committed[id] = fn
		publishMu.Unlock()
		var once sync.Once
		return func() {
			once.Do(func() {
				publishMu.Lock()
				delete(committed, id)
				publishMu.Unlock()
			})
		}
	}
	conflict := &conflictOnce{}
	shared := &backend{
		Backend:  b,
		conflict: conflict,
		publish: func(evs ...*timebox.Event) {
			published := cloneCommittedEvents(evs)
			publishMu.Lock()
			handlers := make([]timebox.Publisher, 0, len(committed))
			for _, fn := range committed {
				handlers = append(handlers, fn)
			}
			publishMu.Unlock()
			for _, fn := range handlers {
				fn(published...)
			}
		},
	}
	flowStore, err := timebox.NewStore(shared, cfg.FlowStoreConfig())
	assert.NoError(t, err)

	testEnv := &TestEngineEnv{
		T:          t,
		MockClient: mockCli,
		Config:     cfg,
		EventHub:   hub,
		flowStore:  flowStore,
		conflict:   conflict,
		flowExec: flowStore.Executor(
			events.NewFlowState, events.FlowAppliers,
		),
		backend:   shared,
		subscribe: subscribe,
		unsubs:    &unsubscribeTracker{},
		ownsHub:   ownsHub,
	}

	deps := mergeDependencies(
		testEnv.engineDeps(scheduler.Now, scheduler.NewTimer), overrides,
	)
	deps.EventHub = hub
	testEnv.Engine, err = engine.New(cfg, deps, testEnv.OpenBackend)
	assert.NoError(t, err)

	testEnv.Cleanup = func() {
		_ = testEnv.Engine.Stop()
		testEnv.unsubscribeAll()
		if testEnv.ownsHub {
			testEnv.EventHub.Close()
		}
		_ = shared.Close()
	}

	return testEnv
}

func raiseFlowEvent(
	ag *timebox.Aggregator[api.FlowState], ev FlowEvent,
) error {
	return events.Raise(ag, ev.Type, ev.Data)
}

func mergeDependencies(
	defaults engine.Dependencies, overrides engine.Dependencies,
) engine.Dependencies {
	if overrides.Scripts != nil {
		defaults.Scripts = overrides.Scripts
	}
	if overrides.Steps != nil {
		defaults.Steps = overrides.Steps
	}
	if overrides.Clock != nil {
		defaults.Clock = overrides.Clock
	}
	if overrides.TimerConstructor != nil {
		defaults.TimerConstructor = overrides.TimerConstructor
	}
	if overrides.EventHub != nil {
		defaults.EventHub = overrides.EventHub
	}
	return defaults
}

func cloneCommittedEvents(src []*timebox.Event) []*timebox.Event {
	res := make([]*timebox.Event, 0, len(src))
	for _, ev := range src {
		if ev == nil {
			res = append(res, nil)
			continue
		}
		data := append([]byte{}, ev.Data...)
		res = append(res, &timebox.Event{
			Timestamp:   ev.Timestamp,
			Sequence:    ev.Sequence,
			Type:        ev.Type,
			AggregateID: ev.AggregateID,
			Data:        data,
		})
	}
	return res
}
