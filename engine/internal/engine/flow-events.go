package engine

import (
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

func (a *flowActor) createEventHandler() timebox.Handler {
	const (
		flowStarted      = timebox.EventType(api.EventTypeFlowStarted)
		flowCompleted    = timebox.EventType(api.EventTypeFlowCompleted)
		flowFailed       = timebox.EventType(api.EventTypeFlowFailed)
		workSucceeded    = timebox.EventType(api.EventTypeWorkSucceeded)
		workFailed       = timebox.EventType(api.EventTypeWorkFailed)
		workNotCompleted = timebox.EventType(api.EventTypeWorkNotCompleted)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowStarted:      timebox.MakeHandler(a.processFlowStarted),
		flowCompleted:    timebox.MakeHandler(a.processFlowCompleted),
		flowFailed:       timebox.MakeHandler(a.processFlowFailed),
		workSucceeded:    timebox.MakeHandler(a.processWorkSucceeded),
		workFailed:       timebox.MakeHandler(a.processWorkFailed),
		workNotCompleted: timebox.MakeHandler(a.processWorkNotCompleted),
	})
}

// processFlowStarted handles a FlowStarted event by finding and starting
// initially ready steps
func (a *flowActor) processFlowStarted(
	_ *timebox.Event, _ api.FlowStartedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		if flowTransitions.IsTerminal(ag.Value().Status) {
			return nil, nil
		}

		var fns enqueued
		for _, stepID := range a.findInitialSteps(ag.Value()) {
			fn, err := a.prepareStep(stepID, ag)
			if err != nil {
				slog.Warn("Failed to prepare step",
					log.StepID(stepID),
					log.Error(err))
				continue
			}
			if fn != nil {
				fns = append(fns, fn)
			}
		}
		return fns, nil
	})
}

func (a *flowActor) processFlowCompleted(
	_ *timebox.Event, _ api.FlowCompletedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		if fn := a.maybeDeactivate(ag.Value()); fn != nil {
			return enqueued{fn}, nil
		}
		return nil, nil
	})
}

func (a *flowActor) processFlowFailed(
	_ *timebox.Event, _ api.FlowFailedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		if fn := a.maybeDeactivate(ag.Value()); fn != nil {
			return enqueued{fn}, nil
		}
		return nil, nil
	})
}

// processWorkSucceeded handles a WorkSucceeded event for a specific work item
func (a *flowActor) processWorkSucceeded(
	_ *timebox.Event, event api.WorkSucceededEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		// Terminal flows only record step completions for audit
		if flowTransitions.IsTerminal(ag.Value().Status) {
			_, err := a.checkStepCompletion(ag, event.StepID)
			if err != nil {
				return nil, err
			}
			if fn := a.maybeDeactivate(ag.Value()); fn != nil {
				return enqueued{fn}, nil
			}
			return nil, nil
		}

		completed, err := a.checkStepCompletion(ag, event.StepID)
		if err != nil || !completed {
			return nil, err
		}

		if err := a.skipPendingUnused(ag); err != nil {
			return nil, err
		}

		// Step completed - check if it was a goal step
		if a.isGoalStep(event.StepID, ag.Value()) {
			if err := a.checkTerminal(ag); err != nil {
				return nil, err
			}
			if fn := a.maybeDeactivate(ag.Value()); fn != nil {
				return enqueued{fn}, nil
			}
			return nil, nil
		}

		// Find and start downstream ready steps
		var fns enqueued
		for _, consumerID := range a.findReadySteps(event.StepID, ag.Value()) {
			fn, err := a.prepareStep(consumerID, ag)
			if err != nil {
				slog.Warn("Failed to prepare step",
					log.StepID(consumerID),
					log.Error(err))
				continue
			}
			if fn != nil {
				fns = append(fns, fn)
			}
		}
		return fns, nil
	})
}

// processWorkFailed handles a WorkFailed event for a specific work item
func (a *flowActor) processWorkFailed(
	_ *timebox.Event, event api.WorkFailedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		fn, err := a.handleStepFailure(ag, event.StepID)
		if err != nil {
			return nil, err
		}
		if fn != nil {
			return enqueued{fn}, nil
		}
		return nil, nil
	})
}

// processWorkNotCompleted handles a WorkNotCompleted event for a specific work
// item
func (a *flowActor) processWorkNotCompleted(
	_ *timebox.Event, event api.WorkNotCompletedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		if flowTransitions.IsTerminal(ag.Value().Status) {
			if fn := a.maybeDeactivate(ag.Value()); fn != nil {
				return enqueued{fn}, nil
			}
			return nil, nil
		}
		err := a.handleWorkNotCompleted(ag, event.StepID, event.Token)
		if err != nil {
			return nil, err
		}
		fn, err := a.handleStepFailure(ag, event.StepID)
		if err != nil {
			return nil, err
		}
		if fn != nil {
			return enqueued{fn}, nil
		}
		return nil, nil
	})
}

// execTransaction executes a function within a flow transaction, handling the
// common pattern of collecting deferred work and executing it after commit
func (a *flowActor) execTransaction(
	fn func(ag *FlowAggregator) (enqueued, error),
) error {
	var fns enqueued
	cmd := func(_ *api.FlowState, ag *FlowAggregator) error {
		var err error
		fns, err = fn(ag)
		return err
	}
	if _, err := a.execFlow(flowKey(a.flowID), cmd); err != nil {
		return err
	}
	fns.exec()
	return nil
}

func (a *flowActor) handleEvent(ev *timebox.Event, handler timebox.Handler) {
	if err := handler(ev); err != nil {
		slog.Error("Failed to handle event",
			log.FlowID(a.flowID),
			slog.String("event_type", string(ev.Type)),
			log.Error(err))
	}
}
