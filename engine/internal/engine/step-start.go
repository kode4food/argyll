package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
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
		when   time.Time
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
	if !policy.StepPending(ex.Status) {
		return fmt.Errorf("%w: %s (status=%s)", ErrStepAlreadyPending,
			stepID, ex.Status)
	}

	st := fl.Plan.Steps[stepID]
	willSkip, err := tx.matchGateWillSkip(st, fl)
	if err != nil {
		return err
	}
	if willSkip {
		return tx.performSkip(stepID, policy.RequiredMatchSkipReason)
	}

	inputs, err := tx.collectStepInputs(st, fl)
	if err != nil {
		return err
	}

	shouldExecute, err := tx.evaluateStepPredicate(st, inputs)
	if err != nil {
		return tx.handlePredicateFailure(stepID, err)
	}
	if !shouldExecute {
		return tx.performSkip(stepID, "predicate returned false")
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

	return nil
}

func (tx *flowTx) performSkip(stepID api.StepID, reason string) error {
	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Reason: reason,
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

func (tx *flowTx) canStartStep(stepID api.StepID, flow api.FlowState) bool {
	ready, _ := tx.newStepEval(stepID, flow, tx.Now()).canStart()
	return ready
}

func (tx *flowTx) canStartStepAt(
	stepID api.StepID, flow api.FlowState, when time.Time,
) (bool, time.Time) {
	return tx.newStepEval(stepID, flow, when).canStart()
}

func (tx *flowTx) findInitialSteps(flow api.FlowState) []api.StepID {
	res := make([]api.StepID, 0, len(flow.Executions))
	for sid, ex := range flow.Executions {
		if !policy.StepPending(ex.Status) {
			continue
		}
		if tx.canStartStep(sid, flow) {
			res = append(res, sid)
		}
	}
	return res
}

func (e *Engine) newStepEval(
	stepID api.StepID, flow api.FlowState, when time.Time,
) *stepEval {
	return &stepEval{
		e:      e,
		flow:   flow,
		stepID: stepID,
		step:   flow.Plan.Steps[stepID],
		when:   when,
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
) (api.Args, error) {
	inputs := api.Args{}
	now := tx.Now()
	ev := tx.newStepEval(step.ID, flow, now)
	anchor, err := ev.requiredReadyAt()
	if err != nil {
		return nil, err
	}

	for name, attr := range step.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}

		if attr.IsConst() {
			inputs[name] = parseDefaultValue(attr.ConstValue())
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
				if attr.OptionalDefault() != "" {
					value := parseDefaultValue(attr.OptionalDefault())
					tx.setStepInput(inputs, step, name, attr, value)
				}
				continue
			}
		}

		values, err := ev.inputValues(name, attr, cutoff)
		if err != nil {
			return nil, err
		}
		val, ok := policy.ResolveInputValue(attr.Collect(), values)
		if !ok {
			if !attr.IsRequired() && attr.OptionalDefault() != "" {
				value := parseDefaultValue(attr.OptionalDefault())
				tx.setStepInput(inputs, step, name, attr, value)
				continue
			}
			continue
		}

		tx.setStepInput(inputs, step, name, attr, val)
	}

	return inputs, nil
}

func (tx *flowTx) setStepInput(
	inputs api.Args, step *api.Step, name api.Name, attr *api.AttributeSpec,
	value any,
) {
	val := tx.mapper.MapInput(step, name, attr, value)
	mapped, _ := step.MappedName(name)
	inputs[mapped] = val
}

func (s *stepEval) canStart() (bool, time.Time) {
	ex := s.flow.Executions[s.stepID]
	if !policy.StepPending(ex.Status) {
		return false, time.Time{}
	}
	if !s.e.areOutputsNeeded(s.stepID, s.flow) {
		return false, time.Time{}
	}

	// If the match gate is closed, allow start so prepareStep can skip it
	if willSkip, _ := s.e.matchGateWillSkip(s.step, s.flow); willSkip {
		return true, time.Time{}
	}

	anchor, err := s.requiredReadyAt()
	if err != nil {
		return false, time.Time{}
	}
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
	fulfilled, fulfilledAt, err := s.inputFulfilledAt(name, attr)
	if err != nil {
		return optionalDecision{}
	}

	if fulfilled {
		at := s.optionalAt(anchor, attr.OptionalDeadline())
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

	providers, _ := providerSummaryFor(s.flow, name)
	if providers.Terminal || attr.OptionalDeadline() <= 0 {
		return s.timeoutDecision(name, attr, s.when)
	}

	at := s.optionalAt(anchor, attr.OptionalDeadline())
	if at.IsZero() {
		return optionalDecision{}
	}
	if !at.After(s.when) {
		return s.timeoutDecision(name, attr, at)
	}
	return optionalDecision{nextAt: at}
}

func (s *stepEval) timeoutDecision(
	name api.Name, attr *api.AttributeSpec, cutoff time.Time,
) optionalDecision {
	if policy.TimeoutCanUseValues(attr.Collect()) {
		if len(valuesUntil(s.flow.AttributeValues(name), cutoff)) > 0 {
			return optionalDecision{ready: true, cutoff: cutoff}
		}
	}
	return optionalDecision{ready: true, fallback: true}
}

func (s *stepEval) optionalAt(anchor time.Time, deadlineMS int64) time.Time {
	if anchor.IsZero() {
		return time.Time{}
	}
	return anchor.Add(time.Duration(deadlineMS) * time.Millisecond)
}

func (s *stepEval) requiredReadyAt() (time.Time, error) {
	anchor := s.flow.CreatedAt
	for name, attr := range s.step.Attributes {
		if !attr.IsRequired() {
			continue
		}

		ok, setAt, err := s.inputFulfilledAt(name, attr)
		if err != nil {
			return time.Time{}, err
		}
		if !ok || setAt.IsZero() {
			return time.Time{}, nil
		}
		if anchor.IsZero() || setAt.After(anchor) {
			anchor = setAt
		}
	}
	return anchor, nil
}

// Matcher returns the engine's shared match predicate evaluator
func (e *Engine) Matcher(cfg *api.ScriptConfig, value any) (bool, error) {
	comp, err := e.scripts.Compile(policy.MatchStep, cfg)
	if err != nil {
		return false, errors.Join(ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(cfg.Language)
	if err != nil {
		return false, errors.Join(ErrPredicateEnvFailed, err)
	}

	matched, err := env.EvaluatePredicate(comp, policy.MatchStep, api.Args{policy.MatchInputName: value})
	if err != nil {
		return false, errors.Join(ErrPredicateEvalFailed, err)
	}
	return matched, nil
}

func (s *stepEval) inputFulfilledAt(
	name api.Name, attr *api.AttributeSpec,
) (bool, time.Time, error) {
	values, err := s.inputValues(name, attr, time.Time{})
	if err != nil {
		return false, time.Time{}, err
	}
	providers, completedAt := providerSummaryFor(s.flow, name)
	if !policy.InputFulfilled(attr.Collect(), len(values), providers) {
		return false, time.Time{}, nil
	}

	switch attr.Collect() {
	case api.InputCollectNone:
		return true, completedAt, nil
	case api.InputCollectLast:
		return true, values[len(values)-1].SetAt, nil
	case api.InputCollectAll:
		return true, lastSetAt(values), nil
	case api.InputCollectSome:
		return true, lastSetAt(values), nil
	default: // api.InputCollectFirst
		return true, values[0].SetAt, nil
	}
}

func (s *stepEval) inputValues(
	name api.Name, attr *api.AttributeSpec, cutoff time.Time,
) ([]*api.AttributeValue, error) {
	values := valuesUntil(s.flow.AttributeValues(name), cutoff)
	matched, _, err := policy.MatchCandidateValues(
		attr, values, s.e.Matcher,
	)
	return matched, err
}

func providerSummaryFor(
	flow api.FlowState, name api.Name,
) (policy.ProviderSummary, time.Time) {
	deps, ok := flow.Plan.Attributes[name]
	if !ok || len(deps.Providers) == 0 {
		return policy.ProviderSummary{
			Terminal:     true,
			AllSucceeded: true,
		}, flow.CreatedAt
	}

	res := policy.ProviderSummary{
		Terminal:     true,
		AllSucceeded: true,
	}
	var completedAt time.Time
	for _, sid := range deps.Providers {
		ex, ok := flow.Executions[sid]
		if !ok {
			res.Terminal = false
			res.AllSucceeded = false
			continue
		}
		if !policy.StepTerminal(ex.Status) {
			res.Terminal = false
		}
		if !policy.StepSucceeded(ex.Status) || !hasValueFrom(flow, name, sid) {
			res.AllSucceeded = false
		}
		if ex.CompletedAt.IsZero() {
			continue
		}
		if completedAt.IsZero() || ex.CompletedAt.After(completedAt) {
			completedAt = ex.CompletedAt
		}
	}
	return res, completedAt
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

func hasValueFrom(flow api.FlowState, name api.Name, sid api.StepID) bool {
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
