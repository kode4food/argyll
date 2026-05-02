package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type (
	stepEval struct {
		e      *Engine
		flow   api.FlowState
		stepID api.StepID
		step   *api.Step
		now    time.Time
	}

	providerSummary struct {
		terminal     bool
		allSucceeded bool
		completedAt  time.Time
	}

	optionalDecision struct {
		ready    bool
		fallback bool
		nextAt   time.Time
		cutoff   time.Time
	}
)

// prepareStep validates and prepares a step to execute within a transaction,
// raising the StepStarted event via aggregator and scheduling work execution
// after commit
func (tx *flowTx) prepareStep(stepID api.StepID) error {
	fl := tx.Value()

	ex := fl.Executions[stepID]
	if ex.Status != api.StepPending {
		return fmt.Errorf("%w: %s (status=%s)", ErrStepAlreadyPending,
			stepID, ex.Status)
	}

	st := fl.Plan.Steps[stepID]
	inputs := tx.collectStepInputs(st, fl)

	shouldExecute, err := tx.evaluateStepPredicate(st, inputs)
	if err != nil {
		return tx.handlePredicateFailure(stepID, err)
	}
	if !shouldExecute {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return err
		}
		return call.Perform(
			tx.checkUnreachable,
			tx.checkTerminal,
			tx.startReadyPendingSteps,
		)
	}

	workItemsList, err := computeWorkItems(st, inputs)
	if err != nil {
		return err
	}
	workItemsMap := map[api.Token]api.Args{}
	for _, workInputs := range workItemsList {
		tkn := api.Token(uuid.New().String())
		workItemsMap[tkn] = workInputs
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID:    tx.flowID,
			StepID:    stepID,
			Inputs:    inputs,
			WorkItems: workItemsMap,
		},
	); err != nil {
		return err
	}

	started, err := tx.startPendingWork(st)
	if err != nil {
		return err
	}

	ex = tx.Value().Executions[stepID]
	if hasReadyPendingDispatch(st, ex, tx.Now()) &&
		!tx.canDispatchLocally(st.ID) {
		if err := tx.raiseDispatchDeferred(st.ID); err != nil {
			return err
		}
	}

	if len(started) > 0 {
		tx.OnSuccess(func(flow api.FlowState, _ []*timebox.Event) {
			if stepHasTimeouts(st) {
				tx.CancelPrefixedTasks(
					timeoutStepPrefix(api.FlowStep{
						FlowID: tx.flowID,
						StepID: st.ID,
					}),
				)
			}
			tx.executeStartedWork(st, inputs, flow.Metadata, started)
		})
	}

	return nil
}

func (tx *flowTx) canStartStep(stepID api.StepID, flow api.FlowState) bool {
	ready, _ := tx.canStartStepAt(stepID, flow, tx.Now())
	return ready
}

func (tx *flowTx) canStartStepAt(
	stepID api.StepID, flow api.FlowState, now time.Time,
) (bool, time.Time) {
	return tx.newStepEval(stepID, flow, now).canStart()
}

func (tx *flowTx) findInitialSteps(flow api.FlowState) []api.StepID {
	res := make([]api.StepID, 0, len(flow.Executions))
	for sid, ex := range flow.Executions {
		if ex.Status != api.StepPending {
			continue
		}
		if tx.canStartStep(sid, flow) {
			res = append(res, sid)
		}
	}
	return res
}

func (e *Engine) newStepEval(
	stepID api.StepID, flow api.FlowState, now time.Time,
) *stepEval {
	return &stepEval{
		e:      e,
		flow:   flow,
		stepID: stepID,
		step:   flow.Plan.Steps[stepID],
		now:    now,
	}
}

func (e *Engine) evaluateStepPredicate(
	step *api.Step, inputs api.Args,
) (bool, error) {
	if step.Predicate == nil {
		return true, nil
	}

	comp, err := e.scripts.Compile(step, step.Predicate)
	if err != nil {
		return false, errors.Join(ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		return false, errors.Join(ErrPredicateEnvFailed, err)
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		return false, errors.Join(ErrPredicateEvalFailed, err)
	}

	return shouldExecute, nil
}

func (tx *flowTx) collectStepInputs(
	step *api.Step, flow api.FlowState,
) api.Args {
	inputs := api.Args{}
	now := tx.Now()
	ev := tx.newStepEval(step.ID, flow, now)
	anchor := ev.requiredReadyAt()

	for name, attr := range step.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}

		if attr.IsConst() {
			inputs[name] = parseDefaultValue(attr.InputDefault())
			continue
		}

		var cutoff time.Time
		if attr.IsOptional() {
			dec := ev.optionalDecisionAt(name, attr, anchor)
			if !dec.ready {
				continue
			}
			cutoff = dec.cutoff
			if dec.fallback {
				if attr.InputDefault() != "" {
					value := parseDefaultValue(attr.InputDefault())
					tx.setStepInput(inputs, step, name, attr, value)
				}
				continue
			}
		}

		val, ok := resolveInputValue(flow, name, attr, cutoff)
		if !ok {
			if !attr.IsRequired() && attr.InputDefault() != "" {
				value := parseDefaultValue(attr.InputDefault())
				tx.setStepInput(inputs, step, name, attr, value)
				continue
			}
			continue
		}

		tx.setStepInput(inputs, step, name, attr, val)
	}

	return inputs
}

func (tx *flowTx) setStepInput(
	inputs api.Args, step *api.Step, name api.Name,
	attr *api.AttributeSpec, value any,
) {
	val := tx.mapper.MapInput(step, name, attr, value)
	mapped, _ := step.MappedName(name)
	inputs[mapped] = val
}

func (s *stepEval) canStart() (bool, time.Time) {
	ex := s.flow.Executions[s.stepID]
	if ex.Status != api.StepPending {
		return false, time.Time{}
	}
	if !s.e.areOutputsNeeded(s.stepID, s.flow) {
		return false, time.Time{}
	}

	anchor := s.requiredReadyAt()
	if anchor.IsZero() {
		return false, time.Time{}
	}

	optReady, nextAt := s.hasOptionalReady(anchor)
	if !optReady {
		return false, nextAt
	}
	return true, time.Time{}
}

func (s *stepEval) hasOptionalReady(anchor time.Time) (bool, time.Time) {
	blocked := false
	var nextAt time.Time

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() {
			continue
		}
		dec := s.optionalDecisionAt(name, attr, anchor)
		if !dec.ready {
			blocked = true
		}
		if !dec.nextAt.IsZero() &&
			(nextAt.IsZero() || dec.nextAt.Before(nextAt)) {
			nextAt = dec.nextAt
		}
	}

	return !blocked, nextAt
}

func (s *stepEval) optionalDecisionAt(
	name api.Name, attr *api.AttributeSpec, anchor time.Time,
) optionalDecision {
	deps, ok := s.flow.Plan.Attributes[name]
	fulfilled, fulfilledAt := s.inputFulfilledAt(name, attr)

	if fulfilled {
		at := s.optionalAt(anchor, attr.InputDeadline())
		if at.IsZero() {
			return optionalDecision{ready: true}
		}

		if !fulfilledAt.IsZero() && fulfilledAt.After(at) {
			return s.timeoutDecision(name, attr, at)
		}
		return optionalDecision{ready: true}
	}

	if !ok || len(deps.Providers) == 0 {
		return optionalDecision{ready: true, fallback: true}
	}

	if attr.InputDeadline() <= 0 {
		return s.timeoutDecision(name, attr, s.now)
	}

	at := s.optionalAt(anchor, attr.InputDeadline())
	if at.IsZero() {
		return optionalDecision{}
	}
	if !at.After(s.now) {
		return s.timeoutDecision(name, attr, at)
	}
	return optionalDecision{nextAt: at}
}

func (s *stepEval) timeoutDecision(
	name api.Name, attr *api.AttributeSpec, cutoff time.Time,
) optionalDecision {
	switch attr.InputCollect() {
	case api.InputCollectLast, api.InputCollectSome:
		if len(valuesUntil(s.flow.AttributeValues(name), cutoff)) > 0 {
			return optionalDecision{ready: true, cutoff: cutoff}
		}
	case api.InputCollectNone:
		if len(valuesUntil(s.flow.AttributeValues(name), cutoff)) > 0 {
			return optionalDecision{ready: true, cutoff: cutoff}
		}
	default:
	}
	return optionalDecision{ready: true, fallback: true}
}

func (s *stepEval) optionalAt(anchor time.Time, deadlineMS int64) time.Time {
	if anchor.IsZero() {
		return time.Time{}
	}
	return anchor.Add(time.Duration(deadlineMS) * time.Millisecond)
}

func (s *stepEval) requiredReadyAt() time.Time {
	anchor := s.flow.CreatedAt
	for name, attr := range s.step.Attributes {
		if !attr.IsRequired() {
			continue
		}

		ok, setAt := s.inputFulfilledAt(name, attr)
		if !ok || setAt.IsZero() {
			return time.Time{}
		}
		if anchor.IsZero() || setAt.After(anchor) {
			anchor = setAt
		}
	}
	return anchor
}

func (s *stepEval) inputFulfilledAt(
	name api.Name, attr *api.AttributeSpec,
) (bool, time.Time) {
	values := s.flow.AttributeValues(name)

	switch attr.InputCollect() {
	case api.InputCollectNone:
		providers := s.providerSummary(name)
		if !providers.terminal || len(values) > 0 {
			return false, time.Time{}
		}
		return true, providers.completedAt
	case api.InputCollectFirst:
		if len(values) == 0 {
			return false, time.Time{}
		}
		return true, values[0].SetAt
	case api.InputCollectLast:
		providers := s.providerSummary(name)
		if len(values) == 0 {
			return false, time.Time{}
		}
		if !providers.terminal {
			return false, time.Time{}
		}
		return true, values[len(values)-1].SetAt
	case api.InputCollectAll:
		providers := s.providerSummary(name)
		if len(values) == 0 {
			return false, time.Time{}
		}
		if !providers.terminal || !providers.allSucceeded {
			return false, time.Time{}
		}
		return true, lastSetAt(values)
	case api.InputCollectSome:
		providers := s.providerSummary(name)
		if len(values) == 0 {
			return false, time.Time{}
		}
		if !providers.terminal {
			return false, time.Time{}
		}
		return true, lastSetAt(values)
	default:
		if len(values) == 0 {
			return false, time.Time{}
		}
		return true, values[0].SetAt
	}
}

func (s *stepEval) providerSummary(name api.Name) providerSummary {
	deps, ok := s.flow.Plan.Attributes[name]
	if !ok || len(deps.Providers) == 0 {
		return providerSummary{}
	}

	res := providerSummary{
		terminal:     true,
		allSucceeded: true,
	}
	missingCompletedAt := false
	for _, sid := range deps.Providers {
		ex, ok := s.flow.Executions[sid]
		if !ok {
			res.terminal = false
			res.allSucceeded = false
			missingCompletedAt = true
			continue
		}
		if !stepTransitions.IsTerminal(ex.Status) {
			res.terminal = false
		}
		if ex.Status != api.StepCompleted || !hasValueFrom(s.flow, name, sid) {
			res.allSucceeded = false
		}
		if ex.CompletedAt.IsZero() {
			missingCompletedAt = true
			continue
		}
		if res.completedAt.IsZero() || ex.CompletedAt.After(res.completedAt) {
			res.completedAt = ex.CompletedAt
		}
	}
	if missingCompletedAt {
		res.completedAt = time.Time{}
	}
	return res
}

func resolveInputValue(
	flow api.FlowState, name api.Name, attr *api.AttributeSpec,
	cutoff time.Time,
) (any, bool) {
	values := valuesUntil(flow.AttributeValues(name), cutoff)
	if len(values) == 0 {
		return nil, false
	}
	switch attr.InputCollect() {
	case api.InputCollectLast:
		return values[len(values)-1].Value, true
	case api.InputCollectAll, api.InputCollectSome:
		res := make([]any, 0, len(values))
		for _, v := range values {
			res = append(res, v.Value)
		}
		return res, true
	case api.InputCollectNone:
		return nil, false
	default:
		return values[0].Value, true
	}
}

func valuesUntil(
	values []*api.AttributeValue, cutoff time.Time,
) []*api.AttributeValue {
	if cutoff.IsZero() {
		return values
	}
	res := make([]*api.AttributeValue, 0, len(values))
	for _, v := range values {
		if v.SetAt.IsZero() || v.SetAt.After(cutoff) {
			continue
		}
		res = append(res, v)
	}
	return res
}

func lastSetAt(values []*api.AttributeValue) time.Time {
	var at time.Time
	for _, v := range values {
		if !v.SetAt.IsZero() && (at.IsZero() || v.SetAt.After(at)) {
			at = v.SetAt
		}
	}
	return at
}

func hasValueFrom(
	flow api.FlowState, name api.Name, sid api.StepID,
) bool {
	for _, v := range flow.AttributeValues(name) {
		if v.Step == sid {
			return true
		}
	}
	return false
}

func parseDefaultValue(value string) any {
	var result any
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil
	}
	return result
}
