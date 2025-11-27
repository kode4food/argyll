package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type (
	flowActor struct {
		*Engine
		flowID api.FlowID
		events chan *timebox.Event
	}

	// deferred represents deferred work to be executed after a transaction
	deferred func()
	enqueued []deferred
)

func (a *flowActor) run() {
	defer a.wg.Done()
	defer a.flows.Delete(a.flowID)

	handler := a.createEventHandler()
	idleTimer := time.NewTimer(100 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case event := <-a.events:
			if err := handler(event); err != nil {
				slog.Error("Failed to handle event",
					slog.Any("flow_id", a.flowID),
					slog.Any("event_type", event.Type),
					slog.Any("error", err))
			}

			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(100 * time.Millisecond)

		case <-idleTimer.C:
			select {
			case event := <-a.events:
				if err := handler(event); err != nil {
					slog.Error("Failed to handle event",
						slog.Any("flow_id", a.flowID),
						slog.Any("event_type", event.Type),
						slog.Any("error", err))
				}
				idleTimer.Reset(100 * time.Millisecond)
			default:
				return
			}

		case <-a.ctx.Done():
			return
		}
	}
}

func (a *flowActor) createEventHandler() timebox.Handler {
	const (
		flowStarted      = timebox.EventType(api.EventTypeFlowStarted)
		workSucceeded    = timebox.EventType(api.EventTypeWorkSucceeded)
		workFailed       = timebox.EventType(api.EventTypeWorkFailed)
		workNotCompleted = timebox.EventType(api.EventTypeWorkNotCompleted)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowStarted:      timebox.MakeHandler(a.processFlowStarted),
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
			fn, err := a.prepareStep(a.ctx, stepID, ag)
			if err != nil {
				slog.Warn("Failed to prepare step",
					slog.Any("step_id", stepID),
					slog.Any("error", err))
				continue
			}
			if fn != nil {
				fns = append(fns, fn)
			}
		}
		return fns, nil
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
			return nil, err
		}

		completed, err := a.checkStepCompletion(ag, event.StepID)
		if err != nil {
			return nil, err
		}
		if !completed {
			return nil, nil
		}

		// Step completed - check if it was a goal step
		if a.isGoalStep(event.StepID, ag.Value()) {
			return nil, a.checkTerminal(ag)
		}

		// Find and start downstream ready steps
		var fns enqueued
		for _, consumerID := range a.findReadySteps(event.StepID, ag.Value()) {
			fn, err := a.prepareStep(a.ctx, consumerID, ag)
			if err != nil {
				slog.Warn("Failed to prepare step",
					slog.Any("step_id", consumerID),
					slog.Any("error", err))
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
		return nil, a.handleStepFailure(ag, event.StepID)
	})
}

// processWorkNotCompleted handles a WorkNotCompleted event for a specific work
// item
func (a *flowActor) processWorkNotCompleted(
	_ *timebox.Event, event api.WorkNotCompletedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) (enqueued, error) {
		if flowTransitions.IsTerminal(ag.Value().Status) {
			return nil, nil
		}
		err := a.handleWorkNotCompleted(ag, event.StepID, event.Token)
		if err != nil {
			return nil, err
		}
		return nil, a.handleStepFailure(ag, event.StepID)
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
	if _, err := a.flowExec.Exec(a.ctx, flowKey(a.flowID), cmd); err != nil {
		return err
	}
	fns.exec()
	return nil
}

// isGoalStep returns true if the step is a goal step
func (a *flowActor) isGoalStep(stepID api.StepID, flow *api.FlowState) bool {
	for _, goalID := range flow.Plan.Goals {
		if goalID == stepID {
			return true
		}
	}
	return false
}

// checkTerminal checks for flow completion or failure
func (a *flowActor) checkTerminal(ag *FlowAggregator) error {
	flow := ag.Value()
	if a.isFlowComplete(flow) {
		result := api.Args{}
		for _, goalID := range flow.Plan.Goals {
			if goal := flow.Executions[goalID]; goal != nil {
				for k, v := range goal.Outputs {
					result[k] = v
				}
			}
		}
		return events.Raise(ag, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: a.flowID,
				Result: result,
			},
		)
	}
	if a.IsFlowFailed(flow) {
		return events.Raise(ag, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: a.flowID,
				Error:  a.getFailureReason(flow),
			},
		)
	}
	return nil
}

// failUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (a *flowActor) failUnreachable(ag *FlowAggregator) error {
	for {
		failedAny := false
		flow := ag.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if a.canStepComplete(stepID, flow) {
				continue
			}

			if err := events.Raise(ag, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Error:  "required input no longer available",
				},
			); err != nil {
				return err
			}
			failedAny = true
		}

		if !failedAny {
			return nil
		}
	}
}

// getFailureReason extracts a failure reason from flow state
func (a *flowActor) getFailureReason(flow *api.FlowState) string {
	for stepID, exec := range flow.Executions {
		if exec.Status == api.StepFailed {
			return fmt.Sprintf("step %s failed: %s", stepID, exec.Error)
		}
	}
	return "flow failed"
}

func (a *flowActor) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !a.hasRequired(stepID, flow) {
		return false
	}
	return a.areOutputsNeeded(stepID, flow)
}

func (a *flowActor) hasRequired(stepID api.StepID, flow *api.FlowState) bool {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.Attributes[name]; !ok {
				return false
			}
		}
	}
	return true
}

// prepareStep validates and prepares a step for execution within a
// transaction, raising the StepStarted event via aggregator and returning a
// deferred function to be executed after transaction commit
func (a *flowActor) prepareStep(
	ctx context.Context, stepID api.StepID, ag *FlowAggregator,
) (deferred, error) {
	flow := ag.Value()

	// Validate step exists and is pending
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return nil, fmt.Errorf("%s: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
	}

	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	// Collect inputs
	inputs := a.collectStepInputs(step, flow.GetAttributes())

	// Evaluate predicate
	fs := FlowStep{FlowID: a.flowID, StepID: stepID}
	if !a.evaluateStepPredicate(ctx, fs, step, inputs) {
		// Predicate failed - skip this step
		if err := events.Raise(ag, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Compute work items
	workItemsList := computeWorkItems(step, inputs)
	workItemsMap := make(map[api.Token]api.Args)
	for _, workInputs := range workItemsList {
		token := api.Token(uuid.New().String())
		workItemsMap[token] = workInputs
	}

	// Raise StepStarted event with work items
	if err := events.Raise(ag, api.EventTypeStepStarted, api.StepStartedEvent{
		FlowID:    a.flowID,
		StepID:    stepID,
		Inputs:    inputs,
		WorkItems: workItemsMap,
	}); err != nil {
		return nil, err
	}

	// Create deferred to execute after transaction commits
	return func() {
		execCtx := &ExecContext{
			engine: a.Engine,
			flowID: a.flowID,
			stepID: stepID,
			step:   step,
			inputs: inputs,
		}

		// Execute work items
		items := ag.Value().Executions[stepID].WorkItems
		execCtx.executeWorkItems(ctx, items)
	}, nil
}

func (e enqueued) exec() {
	for _, fn := range e {
		fn()
	}
}

// handleStepFailure handles common failure logic for processWorkFailed and
// processWorkNotCompleted - checking step completion and propagating failures
func (a *flowActor) handleStepFailure(
	ag *FlowAggregator, stepID api.StepID,
) error {
	if flowTransitions.IsTerminal(ag.Value().Status) {
		_, err := a.checkStepCompletion(ag, stepID)
		return err
	}

	completed, err := a.checkStepCompletion(ag, stepID)
	if err != nil {
		return err
	}

	if completed {
		if err := a.failUnreachable(ag); err != nil {
			return err
		}
		return a.checkTerminal(ag)
	}
	return nil
}

// findInitialSteps finds steps that can start when a flow begins
func (a *flowActor) findInitialSteps(flow *api.FlowState) []api.StepID {
	var ready []api.StepID

	for stepID := range flow.Plan.Steps {
		if a.canStartStep(stepID, flow) {
			ready = append(ready, stepID)
		}
	}

	return ready
}

// findReadySteps finds ready steps among downstream consumers of a completed
// step
func (a *flowActor) findReadySteps(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	var ready []api.StepID

	for _, consumerID := range a.getDownstreamConsumers(stepID, flow) {
		if a.canStartStep(consumerID, flow) {
			ready = append(ready, consumerID)
		}
	}

	return ready
}

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (a *flowActor) checkStepCompletion(
	ag *FlowAggregator, stepID api.StepID,
) (completed bool, err error) {
	exec, ok := ag.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return false, nil
	}

	allDone := true
	hasFailed := false
	var failureError string

	for _, item := range exec.WorkItems {
		switch item.Status {
		case api.WorkSucceeded:
			// continue
		case api.WorkFailed:
			hasFailed = true
			if failureError == "" {
				failureError = item.Error
			}
		case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
			allDone = false
		}
	}

	if !allDone {
		return false, nil
	}

	if hasFailed {
		if failureError == "" {
			failureError = "work item failed"
		}
		err = events.Raise(ag, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Error:  failureError,
			},
		)
		return true, err
	}

	// Step succeeded - set attributes and raise completion
	step := ag.Value().Plan.Steps[stepID]
	outputs := aggregateWorkItemOutputs(exec.WorkItems, step)
	dur := time.Since(exec.StartedAt).Milliseconds()

	for key, value := range outputs {
		if _, ok := ag.Value().Attributes[key]; !ok {
			if err := events.Raise(ag, api.EventTypeAttributeSet,
				api.AttributeSetEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Key:    key,
					Value:  value,
				},
			); err != nil {
				return false, err
			}
		}
	}

	err = events.Raise(ag, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   a.flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	)
	return true, err
}

// handleWorkNotCompleted handles retry decision for a specific work item
func (a *flowActor) handleWorkNotCompleted(
	ag *FlowAggregator, stepID api.StepID, token api.Token,
) error {
	exec, ok := ag.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	workItem, ok := exec.WorkItems[token]
	if !ok || workItem.Status != api.WorkNotCompleted {
		return nil
	}

	step, ok := ag.Value().Plan.Steps[stepID]
	if !ok {
		return nil
	}

	if a.ShouldRetry(step, workItem) {
		nextRetryAt := a.CalculateNextRetry(
			step.WorkConfig, workItem.RetryCount,
		)
		return events.Raise(ag, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      a.flowID,
				StepID:      stepID,
				Token:       token,
				RetryCount:  workItem.RetryCount + 1,
				NextRetryAt: nextRetryAt,
				Error:       workItem.Error,
			},
		)
	}

	// Permanent failure
	return events.Raise(ag, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: a.flowID,
			StepID: stepID,
			Token:  token,
			Error:  workItem.Error,
		},
	)
}

// getDownstreamConsumers returns step IDs that consume any output from the
// given step, using the ExecutionPlan's Attributes dependency map
func (a *flowActor) getDownstreamConsumers(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil
	}

	seen := util.Set[api.StepID]{}
	var consumers []api.StepID

	for name, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}

		deps := flow.Plan.Attributes[name]
		if deps == nil {
			continue
		}

		for _, consumerID := range deps.Consumers {
			if seen.Contains(consumerID) {
				continue
			}
			seen.Add(consumerID)
			consumers = append(consumers, consumerID)
		}
	}

	return consumers
}

func aggregateWorkItemOutputs(
	items api.WorkItems, step *api.Step,
) api.Args {
	completed := make([]*api.WorkState, 0, len(items))
	for _, item := range items {
		if item.Status == api.WorkSucceeded {
			completed = append(completed, item)
		}
	}

	switch len(completed) {
	case 0:
		return nil
	case 1:
		return completed[0].Outputs
	default:
		aggregated := map[api.Name][]map[string]any{}
		var multiArgNames []api.Name
		if step != nil {
			multiArgNames = step.MultiArgNames()
		}

		for _, item := range completed {
			for outputName, outputValue := range item.Outputs {
				entry := map[string]any{}
				for _, argName := range multiArgNames {
					if val, ok := item.Inputs[argName]; ok {
						entry[string(argName)] = val
					}
				}
				entry["value"] = outputValue

				aggregated[outputName] = append(aggregated[outputName], entry)
			}
		}

		outputs := api.Args{}
		for name, values := range aggregated {
			outputs[name] = values
		}
		return outputs
	}
}
