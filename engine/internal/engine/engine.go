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
		workflows    sync.Map // map[workflowID]*workflowActor
		handler      timebox.Handler
	}

	EventConsumer      = topic.Consumer[*timebox.Event]
	Executor           = timebox.Executor[*api.EngineState]
	Aggregator         = timebox.Aggregator[*api.EngineState]
	WorkflowExecutor   = timebox.Executor[*api.WorkflowState]
	WorkflowAggregator = timebox.Aggregator[*api.WorkflowState]
)

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
	e := &Engine{
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
	e.handler = e.createEventHandler()
	return e
}

func (e *Engine) createEventHandler() timebox.Handler {
	workflowStarted := timebox.MakeHandler(e.handleWorkflowStarted)
	workflowCompleted := timebox.MakeHandler(e.handleWorkflowCompleted)
	workflowFailed := timebox.MakeHandler(e.handleWorkflowFailed)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		api.EventTypeWorkflowStarted:   workflowStarted,
		api.EventTypeWorkflowCompleted: workflowCompleted,
		api.EventTypeWorkflowFailed:    workflowFailed,
	})
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
	existing, err := e.GetWorkflowState(ctx, flowID)
	if err == nil && existing.ID != "" {
		return ErrWorkflowExists
	}

	if err := plan.ValidateInputs(initState); err != nil {
		return err
	}

	if err := e.scripts.CompilePlan(plan); err != nil {
		return err
	}

	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkflowStartedEvent{
			FlowID:   flowID,
			Plan:     plan,
			Init:     initState,
			Metadata: meta,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowStarted, ev)
		return nil
	}

	_, err = e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
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
	for {
		select {
		case <-e.ctx.Done():
			return

		case event, ok := <-e.consumer.Receive():
			if !ok {
				return
			}
			e.routeEvent(event)
		}
	}
}

func (e *Engine) routeEvent(event *timebox.Event) {
	if err := e.handler(event); err != nil {
		slog.Error("Failed to handle workflow lifecycle event",
			slog.Any("event_type", event.Type),
			slog.Any("error", err))
	}

	if !events.IsWorkflowEvent(event) {
		return
	}

	flowID := event.AggregateID[1]

	actor, loaded := e.workflows.Load(flowID)
	if !loaded {
		wa := &workflowActor{
			Engine: e,
			flowID: flowID,
			events: make(chan *timebox.Event, 100),
		}
		wa.eventHandler = wa.createEventHandler()
		actor, loaded = e.workflows.LoadOrStore(flowID, wa)
		if !loaded {
			e.wg.Add(1)
			go wa.run()
		}
	}

	actor.(*workflowActor).events <- event
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

func (e *Engine) handleWorkflowStarted(
	_ *timebox.Event, data api.WorkflowStartedEvent,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		evData, err := json.Marshal(api.WorkflowActivatedEvent{
			FlowID: data.FlowID,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowActivated, evData)
		return nil
	}

	ctx := context.Background()
	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}

func (e *Engine) handleWorkflowCompleted(
	_ *timebox.Event, data api.WorkflowCompletedEvent,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		evData, err := json.Marshal(api.WorkflowDeactivatedEvent{
			FlowID: data.FlowID,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowDeactivated, evData)
		return nil
	}

	ctx := context.Background()
	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}

func (e *Engine) handleWorkflowFailed(
	_ *timebox.Event, data api.WorkflowFailedEvent,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		evData, err := json.Marshal(api.WorkflowDeactivatedEvent{
			FlowID: data.FlowID,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowDeactivated, evData)
		return nil
	}

	ctx := context.Background()
	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}
