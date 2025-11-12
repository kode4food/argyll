package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/client"
	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	Engine struct {
		stepClient   client.Client
		ctx          context.Context
		consumer     EventConsumer
		engineExec   *Executor
		workflowExec *WorkflowExecutor
		config       *config.Config
		cancel       context.CancelFunc
		scripts      *ScriptRegistry
		wg           sync.WaitGroup
	}

	EventConsumer      = topic.Consumer[*timebox.Event]
	Executor           = timebox.Executor[*api.EngineState]
	Aggregator         = timebox.Aggregator[*api.EngineState]
	WorkflowExecutor   = timebox.Executor[*api.WorkflowState]
	WorkflowAggregator = timebox.Aggregator[*api.WorkflowState]
)

const doneChannelBufferSize = 100

var (
	ErrShutdownTimeout      = errors.New("shutdown timeout exceeded")
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrWorkflowExists       = errors.New("workflow exists")
	ErrStepNotFound         = errors.New("step not found")
	ErrStepExists           = errors.New("step exists")
	ErrScriptCompileFailed  = errors.New("failed to compile scripts for plan")
	ErrExecutionPlanMissing = errors.New("execution plan missing required data")
	ErrStepNotInPlan        = errors.New("step not in execution plan")
	ErrInvalidTransition    = errors.New("invalid step status transition")
	ErrAttributeAlreadySet  = errors.New("attribute already set")
)

func New(
	engine, workflow *timebox.Store, client client.Client, hub timebox.EventHub,
	cfg *config.Config,
) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		engineExec: timebox.NewExecutor(
			engine, events.NewEngineState, events.EngineAppliers,
		),
		workflowExec: timebox.NewExecutor(
			workflow, events.NewWorkflowState, events.WorkflowAppliers,
		),
		stepClient: client,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		consumer:   hub.NewConsumer(),
		scripts:    NewScriptRegistry(),
	}
}

func (e *Engine) Start() {
	slog.Info("Engine starting")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.RecoverWorkflows(ctx); err != nil {
		slog.Error("Failed to recover workflows",
			slog.Any("error", err))
	}

	go e.eventLoop()
	go e.retryLoop()
}

func (e *Engine) Stop() error {
	e.cancel()
	defer e.consumer.Close()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.saveEngineSnapshot()
		slog.Info("Engine stopped")
		return nil
	case <-time.After(e.config.ShutdownTimeout):
		return ErrShutdownTimeout
	}
}

func (e *Engine) StartWorkflow(
	ctx context.Context, flowID timebox.ID, plan *api.ExecutionPlan,
	initState api.Args, meta api.Metadata,
) error {
	return e.startWorkflow(ctx, flowID, plan, initState, meta)
}

func (e *Engine) UnregisterStep(ctx context.Context, stepID timebox.ID) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		ev, err := json.Marshal(api.StepUnregisteredEvent{StepID: stepID})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeStepUnregistered, ev)
		return nil
	}

	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}

func (e *Engine) GetEngineState(ctx context.Context) (*api.EngineState, error) {
	return e.engineExec.Exec(ctx, events.EngineID,
		func(st *api.EngineState, ag *Aggregator) error {
			return nil
		},
	)
}

func (e *Engine) ListSteps(ctx context.Context) ([]*api.Step, error) {
	engState, err := e.GetEngineState(ctx)
	if err != nil {
		return nil, err
	}

	var steps []*api.Step
	for _, step := range engState.Steps {
		steps = append(steps, step)
	}

	return steps, nil
}

func (e *Engine) eventLoop() {
	pending := map[timebox.ID]bool{}
	done := make(chan timebox.ID, doneChannelBufferSize)

	for {
		select {
		case <-e.ctx.Done():
			return

		case event, ok := <-e.consumer.Receive():
			if !ok {
				return
			}
			e.handleWorkflowEvent(event, pending, done)

		case flowID := <-done:
			delete(pending, flowID)
		}
	}
}

func (e *Engine) retryLoop() {
	ticker := time.NewTicker(e.config.RetryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkPendingRetries()
		}
	}
}

func (e *Engine) checkPendingRetries() {
	ctx := context.Background()

	engineState, err := e.GetEngineState(ctx)
	if err != nil {
		slog.Error("Failed to get engine state",
			slog.Any("error", err))
		return
	}

	now := time.Now()
	for flowID := range engineState.ActiveWorkflows {
		flow, err := e.GetWorkflowState(ctx, flowID)
		if err != nil {
			continue
		}

		for stepID, exec := range flow.Executions {
			if exec.WorkItems == nil {
				continue
			}

			for token, workItem := range exec.WorkItems {
				if workItem.Status == api.WorkPending &&
					!workItem.NextRetryAt.IsZero() &&
					workItem.NextRetryAt.Before(now) {
					slog.Debug("Retrying work",
						slog.Any("flow_id", flowID),
						slog.Any("step_id", stepID),
						slog.Any("token", token),
						slog.Int("retry_count", workItem.RetryCount))

					step := flow.Plan.GetStep(stepID)
					if step != nil {
						e.retryWork(ctx, flowID, stepID, step, workItem.Inputs)
					}
				}
			}
		}
	}
}

func (e *Engine) retryWork(
	ctx context.Context, flowID, stepID timebox.ID, step *api.Step,
	inputs api.Args,
) {
	execCtx := &ExecContext{
		start:  time.Now(),
		engine: e,
		step:   step,
		inputs: inputs,
		flowID: flowID,
		stepID: stepID,
	}

	execCtx.executeWorkItem(ctx, inputs)
}

func (e *Engine) handleWorkflowEvent(
	event *timebox.Event, pending map[timebox.ID]bool, done chan timebox.ID,
) {
	if !events.IsWorkflowEvent(event) {
		return
	}

	flowID := event.AggregateID[1]

	// Apply workflow lifecycle events to engine state for recovery tracking
	switch event.Type {
	case api.EventTypeWorkflowStarted,
		api.EventTypeWorkflowCompleted,
		api.EventTypeWorkflowFailed:
		e.applyLifecycleEvent(event)
	}

	// Trigger workflow processing for state-changing events
	switch event.Type {
	case api.EventTypeWorkflowStarted,
		api.EventTypeStepCompleted,
		api.EventTypeStepFailed,
		api.EventTypeStepSkipped,
		api.EventTypeAttributeSet,
		api.EventTypeWorkCompleted,
		api.EventTypeWorkFailed:
		e.maybeProcessWorkflow(flowID, pending, done)
	}
}

func (e *Engine) applyLifecycleEvent(event *timebox.Event) {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		// Raise the event on the engine aggregate so it's properly persisted
		ag.Raise(event.Type, event.Data)
		return nil
	}

	_, err := e.engineExec.Exec(e.ctx, events.EngineID, cmd)
	if err != nil {
		slog.Error("Failed to apply event",
			slog.Any("event_type", event.Type),
			slog.Any("error", err))
	}
}

func (e *Engine) maybeProcessWorkflow(
	flowID timebox.ID, pending map[timebox.ID]bool, done chan timebox.ID,
) {
	if pending[flowID] {
		return
	}
	pending[flowID] = true
	go func(flowID timebox.ID) {
		e.processWorkflow(flowID)
		done <- flowID
	}(flowID)
}

func (e *Engine) getCompiledFromPlan(
	flowID, stepID timebox.ID, getter func(*api.StepInfo) (any, error),
) (any, error) {
	flow, err := e.GetWorkflowState(e.ctx, flowID)
	if err != nil {
		return nil, err
	}

	if !e.ensureScriptsCompiled(flowID, flow) {
		return nil, ErrScriptCompileFailed
	}

	info, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil, ErrStepNotInPlan
	}

	return getter(info)
}

func (e *Engine) GetCompiledPredicate(flowID, stepID timebox.ID) (any, error) {
	return e.getCompiledFromPlan(flowID, stepID,
		func(info *api.StepInfo) (any, error) {
			if info.Predicate == nil {
				return nil, fmt.Errorf("%w: predicate", ErrExecutionPlanMissing)
			}
			return info.Predicate, nil
		},
	)
}

func (e *Engine) GetCompiledScript(flowID, stepID timebox.ID) (any, error) {
	return e.getCompiledFromPlan(
		flowID, stepID, func(info *api.StepInfo,
		) (any, error) {
			if info.Script == nil {
				return nil, fmt.Errorf("%w: script", ErrExecutionPlanMissing)
			}
			return info.Script, nil
		},
	)
}

func (e *Engine) saveEngineSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.engineExec.SaveSnapshot(ctx, events.EngineID); err != nil {
		slog.Error("Failed to save engine snapshot",
			slog.Any("error", err))
		return
	}
	slog.Info("Engine snapshot saved")
}
