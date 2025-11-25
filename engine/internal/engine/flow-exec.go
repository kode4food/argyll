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

// Event types that should wake up the flow actor
var flowProcessingEvents = util.SetOf(
	timebox.EventType(api.EventTypeFlowStarted),
	timebox.EventType(api.EventTypeStepCompleted),
	timebox.EventType(api.EventTypeStepFailed),
	timebox.EventType(api.EventTypeStepSkipped),
	timebox.EventType(api.EventTypeWorkSucceeded),
	timebox.EventType(api.EventTypeWorkFailed),
	timebox.EventType(api.EventTypeWorkNotCompleted),
	timebox.EventType(api.EventTypeRetryScheduled),
)

func (a *flowActor) run() {
	defer a.wg.Done()
	defer a.flows.Delete(a.flowID)

	idleTimer := time.NewTimer(100 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case event := <-a.events:
			// Only process if it's a relevant event type
			if !flowProcessingEvents.Contains(event.Type) {
				continue
			}

			// Drain all pending events (coalesce them)
			a.drainEvents()

			// Process flow once to handle all accumulated state changes
			a.processFlow()

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
				if !flowProcessingEvents.Contains(event.Type) {
					idleTimer.Reset(100 * time.Millisecond)
					continue
				}
				a.drainEvents()
				a.processFlow()
				idleTimer.Reset(100 * time.Millisecond)
			default:
				return
			}

		case <-a.ctx.Done():
			return
		}
	}
}

// drainEvents drains all pending events from the channel, coalescing them
func (a *flowActor) drainEvents() {
	for {
		select {
		case <-a.events:
			// Just drain, don't process individually
		default:
			return
		}
	}
}

func (a *flowActor) processFlow() {
	flow, err := a.GetFlowState(a.ctx, a.flowID)
	if err != nil {
		slog.Error("Failed to get flow state",
			slog.Any("flow_id", a.flowID),
			slog.Any("error", err))
		return
	}

	if !a.ensureScriptsCompiled(flow) {
		return
	}

	// Execute all state transitions in single transaction
	enqueued, err := a.processFlowTransaction()
	if err != nil {
		slog.Error("Failed to process flow transaction",
			slog.Any("flow_id", a.flowID),
			slog.Any("error", err))
		return
	}

	// Transaction succeeded - execute deferred work
	enqueued.exec()
}

// processFlowTransaction executes all flow processing in a single transaction,
// returning deferred functions to execute after successful commit. Terminal
// flows still process to record step completions for audit/compensation
func (a *flowActor) processFlowTransaction() (enqueued, error) {
	var fns enqueued

	cmd := func(_ *api.FlowState, ag *FlowAggregator) error {
		// Terminal flows only record pending step completions
		if flowTransitions.IsTerminal(ag.Value().Status) {
			return a.checkCompletableSteps(ag)
		}

		// Loop until no new events are raised (handles cascading failures)
		for {
			before := len(ag.Enqueued())

			// 1. Evaluate flow state (skip/fail steps)
			if err := a.evaluateFlowState(ag); err != nil {
				return err
			}

			// 2. Handle work items that didn't complete (retry decisions)
			if err := a.handleWorkNotCompleted(ag); err != nil {
				return err
			}

			// 3. Check completable steps
			if err := a.checkCompletableSteps(ag); err != nil {
				return err
			}

			// No new events raised, done evaluating
			if len(ag.Enqueued()) == before {
				break
			}
		}

		// 4. Find ready steps
		ready := a.findReadySteps(ag.Value())

		// 5. Prepare ready steps and accumulate work
		for _, stepID := range ready {
			fn, err := a.prepareStep(a.ctx, stepID, ag)
			if err != nil {
				// Log but continue with other steps
				slog.Warn("Failed to prepare step in transaction",
					slog.Any("step_id", stepID),
					slog.Any("error", err))
				continue
			}

			if fn != nil {
				fns = append(fns, fn)
			}
		}

		// 6. Handle terminal state if no ready steps
		if len(ready) == 0 {
			flow := ag.Value()
			if a.isFlowComplete(flow) {
				// Flow completed successfully
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
			} else if a.IsFlowFailed(flow) {
				// Flow failed
				return events.Raise(ag, api.EventTypeFlowFailed,
					api.FlowFailedEvent{
						FlowID: a.flowID,
						Error:  a.getFlowFailureReason(flow),
					},
				)
			}
		}

		return nil
	}

	// Execute transaction
	_, err := a.flowExec.Exec(a.ctx, flowKey(a.flowID), cmd)
	if err != nil {
		return nil, err
	}

	return fns, nil
}

// evaluateFlowState evaluates flow state and raises skip/fail events within
// the provided transaction context via the aggregator
func (a *flowActor) evaluateFlowState(ag *FlowAggregator) error {
	for stepID := range ag.Value().Plan.Steps {
		exec, ok := ag.Value().Executions[stepID]
		if !ok || exec.Status != api.StepPending {
			continue
		}

		if err := a.maybeSkipStep(ag, stepID); err != nil {
			return err
		}
	}

	return nil
}

// handleWorkNotCompleted handles work items that didn't complete making retry
// or fail decisions
func (a *flowActor) handleWorkNotCompleted(ag *FlowAggregator) error {
	for stepID, exec := range ag.Value().Executions {
		if exec.Status != api.StepActive {
			continue
		}

		step := ag.Value().Plan.GetStep(stepID)
		if step == nil {
			continue
		}

		for token, workItem := range exec.WorkItems {
			if workItem.Status != api.WorkNotCompleted {
				continue
			}

			// Make retry decision
			if a.ShouldRetry(step, workItem) {
				// Raise RetryScheduled event
				nextRetryAt := a.CalculateNextRetry(
					step.WorkConfig, workItem.RetryCount,
				)
				if err := events.Raise(ag, api.EventTypeRetryScheduled,
					api.RetryScheduledEvent{
						FlowID:      a.flowID,
						StepID:      stepID,
						Token:       token,
						RetryCount:  workItem.RetryCount + 1,
						NextRetryAt: nextRetryAt,
						Error:       workItem.Error,
					},
				); err != nil {
					return err
				}
			} else {
				// Raise WorkFailed event (permanent failure)
				if err := events.Raise(ag, api.EventTypeWorkFailed,
					api.WorkFailedEvent{
						FlowID: a.flowID,
						StepID: stepID,
						Token:  token,
						Error:  workItem.Error,
					},
				); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// checkCompletableSteps checks for completable steps and raises completion or
// failure events via the aggregator
func (a *flowActor) checkCompletableSteps(ag *FlowAggregator) error {
	for stepID, exec := range ag.Value().Executions {
		if exec.Status != api.StepActive {
			continue
		}

		allDone := true
		hasFailed := false
		var failureError string

		for _, item := range exec.WorkItems {
			switch item.Status {
			case api.WorkSucceeded:
				// Work succeeded, continue
			case api.WorkFailed:
				// Work failed permanently
				hasFailed = true
				if failureError == "" {
					failureError = item.Error
				}
			case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
				// Not done yet (not completed, pending retry, or still active)
				allDone = false
			}
		}

		if !allDone {
			continue
		}

		if hasFailed {
			if failureError == "" {
				failureError = "work item failed"
			}

			// Raise StepFailed event via aggregator
			if err := events.Raise(ag, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Error:  failureError,
				},
			); err != nil {
				return err
			}
		} else {
			step := ag.Value().Plan.GetStep(stepID)
			outputs := aggregateWorkItemOutputs(exec.WorkItems, step)
			dur := time.Since(exec.StartedAt).Milliseconds()

			// Raise AttributeSet events for each output
			for key, value := range outputs {
				// Check if attribute is already set
				if _, ok := ag.Value().Attributes[key]; !ok {
					if err := events.Raise(ag, api.EventTypeAttributeSet,
						api.AttributeSetEvent{
							FlowID: a.flowID,
							StepID: stepID,
							Key:    key,
							Value:  value,
						},
					); err != nil {
						return err
					}
				}
			}

			// Raise StepCompleted event via aggregator
			if err := events.Raise(ag, api.EventTypeStepCompleted,
				api.StepCompletedEvent{
					FlowID:   a.flowID,
					StepID:   stepID,
					Outputs:  outputs,
					Duration: dur,
				},
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// maybeSkipStep checks if a step should be skipped or failed, raising
// appropriate events via the aggregator
func (a *flowActor) maybeSkipStep(ag *FlowAggregator, stepID api.StepID) error {
	flow := ag.Value()
	if a.canStepComplete(stepID, flow) {
		if !a.areOutputsNeeded(stepID, flow) {
			// Raise StepSkipped event via aggregator
			return events.Raise(ag, api.EventTypeStepSkipped,
				api.StepSkippedEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Reason: "outputs not needed",
				},
			)
		}
		return nil
	}

	// Raise StepFailed event via aggregator
	return events.Raise(ag, api.EventTypeStepFailed, api.StepFailedEvent{
		FlowID: a.flowID,
		StepID: stepID,
		Error:  "required inputs cannot be satisfied",
	})
}

// getFlowFailureReason extracts a failure reason from flow state
func (a *flowActor) getFlowFailureReason(flow *api.FlowState) string {
	for stepID, exec := range flow.Executions {
		if exec.Status == api.StepFailed {
			return fmt.Sprintf("step %s failed: %s", stepID, exec.Error)
		}
	}
	return "flow failed"
}

func (a *flowActor) findReadySteps(flow *api.FlowState) []api.StepID {
	visited := util.Set[api.StepID]{}
	var ready []api.StepID

	for _, goalID := range flow.Plan.Goals {
		a.findReadyStepsFromGoal(goalID, flow, visited, &ready)
	}

	return ready
}

func (a *flowActor) findReadyStepsFromGoal(
	stepID api.StepID, flow *api.FlowState, visited util.Set[api.StepID],
	ready *[]api.StepID,
) {
	if visited.Contains(stepID) {
		return
	}
	visited.Add(stepID)

	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepPending {
		return
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return
	}

	for name, attr := range step.Attributes {
		if !attr.IsRequired() {
			continue
		}

		if _, hasAttr := flow.Attributes[name]; hasAttr {
			continue
		}

		deps := flow.Plan.Attributes[name]
		if deps == nil || len(deps.Providers) == 0 {
			continue
		}

		for _, providerID := range deps.Providers {
			a.findReadyStepsFromGoal(providerID, flow, visited, ready)
		}
	}

	if a.isStepReadyForExec(stepID, flow) {
		*ready = append(*ready, stepID)
	}
}

func (a *flowActor) isStepReadyForExec(
	stepID api.StepID, flow *api.FlowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !a.isStepReady(stepID, flow) {
		return false
	}
	return a.areOutputsNeeded(stepID, flow)
}

func (a *flowActor) isStepReady(stepID api.StepID, flow *api.FlowState) bool {
	step := flow.Plan.GetStep(stepID)
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

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return nil, fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	// Collect inputs
	inputs := a.collectStepInputs(step, flow.GetAttributeArgs())

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

func aggregateWorkItemOutputs(
	items map[api.Token]*api.WorkState, step *api.Step,
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
