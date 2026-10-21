package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

type (
	stepEval struct {
		when   time.Time
		e      *Engine
		step   *api.Step
		flow   *api.FlowState
		stepID api.StepID
	}

	optionalDecision struct {
		nextAt   time.Time
		values   []*api.AttributeValue
		ready    bool
		fallback bool
	}
)

var (
	ErrPredicateFalse = errors.New("predicate returned false")
)

// Matcher returns the engine's shared match predicate evaluator
func (e *Engine) Matcher(cfg *api.ScriptConfig, value any) (bool, error) {
	comp, err := e.scripts.Compile(script.MatchStep, cfg)
	if err != nil {
		return false, errors.Join(ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(cfg.Language)
	if err != nil {
		return false, errors.Join(ErrScriptEnvFailed, err)
	}

	matched, err := env.EvaluateMatch(comp, value)
	if err != nil {
		return false, errors.Join(ErrMatchEvalFailed, err)
	}
	return matched, nil
}

// prepareStep validates a step and raises its StepStarted event with the work
// items to run, leaving dispatch to the work start path
func (tx *flowTx) prepareStep(sid api.StepID) error {
	fl := tx.Value()

	ex := fl.Executions[sid]
	if !policy.StepPending(ex.Status) {
		return fmt.Errorf("%w: %s (status=%s)", ErrStepAlreadyPending,
			sid, ex.Status)
	}

	st := fl.Plan.Steps[sid]
	unsatisfied, err := tx.matchGateUnsatisfiedInputs(st, &fl)
	if err != nil {
		return tx.handlePredicateFailure(sid, ex.Inputs, err)
	}
	if len(unsatisfied) > 0 {
		inputs, err := tx.collectStepInputs(st, &fl)
		if err != nil {
			return tx.handlePredicateFailure(sid, ex.Inputs, err)
		}
		return tx.performSkip(
			sid, policy.RequiredMatchSkipReason, inputs, unsatisfied,
		)
	}

	inputs, err := tx.collectStepInputs(st, &fl)
	if err != nil {
		return tx.handlePredicateFailure(sid, ex.Inputs, err)
	}

	workItemsList, err := computeWorkItems(st, inputs)
	if err != nil {
		return err
	}
	workItemsMap := map[api.Token]api.Args{}
	for _, workInputs := range workItemsList {
		shouldExecute, err := tx.evaluateStepPredicate(
			st, inputs.Apply(workInputs),
		)
		if err != nil {
			return tx.handlePredicateFailure(sid, inputs, err)
		}
		if !shouldExecute {
			continue
		}
		tkn := api.Token(uuid.New().String())
		workItemsMap[tkn] = workInputs
	}
	if len(workItemsList) > 0 && len(workItemsMap) == 0 {
		return tx.performSkip(sid, ErrPredicateFalse.Error(), inputs, nil)
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID:    tx.flowID,
			StepID:    sid,
			Inputs:    inputs,
			WorkItems: workItemsMap,
		},
	); err != nil {
		return err
	}

	return tx.continueStepWork(sid)
}

func (tx *flowTx) performSkip(
	sid api.StepID, reason string, inputs api.Args, unsatisfied []api.Name,
) error {
	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			FlowID:      tx.flowID,
			StepID:      sid,
			Reason:      reason,
			Inputs:      inputs,
			Unsatisfied: unsatisfied,
		},
	); err != nil {
		return err
	}
	return tx.advanceFlow()
}

func (tx *flowTx) canStartStepAt(
	sid api.StepID, fl *api.FlowState, when time.Time,
) (bool, time.Time) {
	return tx.newStepEval(sid, fl, when).canStart()
}

func (e *Engine) newStepEval(
	sid api.StepID, fl *api.FlowState, when time.Time,
) *stepEval {
	return &stepEval{
		e:      e,
		flow:   fl,
		stepID: sid,
		step:   fl.Plan.Steps[sid],
		when:   when,
	}
}

func (e *Engine) evaluateStepPredicate(
	st *api.Step, inputs api.Args,
) (bool, error) {
	if st.Predicate == nil {
		return true, nil
	}

	comp, err := e.scripts.Compile(st, st.Predicate)
	if err != nil {
		return false, errors.Join(ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(st.Predicate.Language)
	if err != nil {
		return false, errors.Join(ErrScriptEnvFailed, err)
	}

	shouldExecute, err := env.EvaluatePredicate(comp, st, inputs)
	if err != nil {
		return false, errors.Join(ErrPredicateEvalFailed, err)
	}

	return shouldExecute, nil
}

func (tx *flowTx) collectStepInputs(
	st *api.Step, fl *api.FlowState,
) (api.Args, error) {
	inputs := api.Args{}
	ev := tx.newStepEval(st.ID, fl, tx.Now())
	anchor, err := ev.requiredReadyAt()
	if err != nil {
		return nil, err
	}

	for name, attr := range st.Attributes {
		if attr.IsConst() {
			inputs[name] = parseDefaultValue(attr.ConstValue())
			continue
		}
		if !attr.IsInput() {
			continue
		}

		var values []*api.AttributeValue
		if attr.IsOptional() {
			dec := ev.optionalDecisionAt(name, attr, anchor)
			if !dec.ready {
				continue
			}
			if dec.fallback && attr.OptionalDefault() != "" {
				tx.setStepInput(setStepInputArgs{
					inputs: inputs,
					step:   st,
					name:   name,
					attr:   attr,
					value:  parseDefaultValue(attr.OptionalDefault()),
				})
			}
			if dec.fallback {
				continue
			}
			values = dec.values
		} else {
			values, err = ev.inputValues(name, attr)
			if err != nil {
				return nil, err
			}
		}
		val, ok := policy.ResolveInputValue(attr.Collect(), values)
		if !ok {
			continue
		}
		tx.setStepInput(setStepInputArgs{
			inputs: inputs,
			step:   st,
			name:   name,
			attr:   attr,
			value:  val,
		})
	}
	return inputs, nil
}

type setStepInputArgs struct {
	inputs api.Args
	step   *api.Step
	name   api.Name
	attr   *api.AttributeSpec
	value  any
}

func (tx *flowTx) setStepInput(args setStepInputArgs) {
	val := tx.mapper.MapInput(args.step, args.name, args.attr, args.value)
	mapped, _ := args.step.MappedName(args.name)
	args.inputs[mapped] = val
}

func (s *stepEval) canStart() (bool, time.Time) {
	ex := s.flow.Executions[s.stepID]
	if !policy.StepPending(ex.Status) {
		return false, time.Time{}
	}
	if !s.e.areOutputsNeeded(s.stepID, s.flow, startGoals(s.flow), s.when) {
		return false, time.Time{}
	}

	// Let preparation record a closed or broken match gate's outcome
	if willSkip, err := s.e.matchGateWillSkip(s.step, s.flow); willSkip ||
		err != nil {
		return true, time.Time{}
	}

	anchor, err := s.requiredReadyAt()
	if err != nil {
		return true, time.Time{}
	}
	if anchor.At.IsZero() {
		return false, time.Time{}
	}

	optReady, nextAt := s.hasOptionalReady(anchor)
	if !optReady {
		return false, nextAt
	}
	return true, time.Time{}
}

func (s *stepEval) hasOptionalReady(
	anchor api.Completion,
) (bool, time.Time) {
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

func (s *stepEval) hasOptionalDecision(
	name api.Name, attr *api.AttributeSpec,
) bool {
	anchor, err := s.requiredReadyAt()
	if err != nil {
		return false
	}
	return s.optionalDecisionAt(name, attr, anchor).ready
}

func (s *stepEval) optionalDecisionAt(
	name api.Name, attr *api.AttributeSpec, anchor api.Completion,
) optionalDecision {
	if anchor.At.IsZero() {
		return optionalDecision{}
	}
	values, err := s.inputValues(name, attr)
	if err != nil {
		return optionalDecision{}
	}
	if attr.OptionalDeadline() == 0 {
		return s.resolveImmediateOptional(name, attr, values, anchor)
	}
	values, at := policy.CollectWindow(
		anchor.At, attr.OptionalDeadline(), values,
	)
	providers, last := providerSummaryFor(s.flow, name)
	if policy.InputFulfilled(
		attr.Collect(), len(values), providers, s.flow.HasInit(name),
	) {
		readyAt := policy.FulfilledAt(attr.Collect(), values, last.At)
		if readyAt.Before(at) {
			return optionalDecision{ready: true, values: values}
		}
	}
	if providers.Terminal || !at.After(s.when) {
		return optionalDecision{ready: true, fallback: true}
	}
	if attr.Collect() == api.InputCollectNone && len(values) > 0 {
		return optionalDecision{ready: true, fallback: true}
	}
	if attr.Collect() == api.InputCollectAll && !canCollectAll(name, s.flow) {
		return optionalDecision{ready: true, fallback: true}
	}
	return optionalDecision{nextAt: at}
}

func (s *stepEval) resolveImmediateOptional(
	name api.Name, attr *api.AttributeSpec, values []*api.AttributeValue,
	anchor api.Completion,
) optionalDecision {
	accepted := make([]*api.AttributeValue, 0, len(values))
	// Init values have no producing execution, so their sequence is zero
	for _, value := range values {
		if s.flow.Executions[value.Step].Completion.Within(anchor.Seq) {
			accepted = append(accepted, value)
		}
	}
	providers, _ := providerSummaryAt(s.flow, name, anchor.Seq)
	if policy.InputFulfilled(
		attr.Collect(), len(accepted), providers, s.flow.HasInit(name),
	) {
		return optionalDecision{ready: true, values: accepted}
	}
	return optionalDecision{ready: true, fallback: true}
}

// requiredReadyAt returns when the required inputs became satisfied, or zero
// while any is unsatisfied. collect:first counts from its first value
func (s *stepEval) requiredReadyAt() (api.Completion, error) {
	res := api.Completion{At: s.flow.CreatedAt}
	for name, attr := range s.step.Attributes {
		if !attr.IsRequired() {
			continue
		}
		values, err := s.inputValues(name, attr)
		if err != nil {
			return api.Completion{}, err
		}
		providers, last := providerSummaryFor(s.flow, name)
		if !policy.InputFulfilled(
			attr.Collect(), len(values), providers, s.flow.HasInit(name),
		) {
			return api.Completion{}, nil
		}
		if attr.Collect() == api.InputCollectFirst {
			last = s.flow.Executions[values[0].Step].Completion
		}
		res = res.Latest(last)
	}
	return res, nil
}

func (s *stepEval) inputValues(
	name api.Name, attr *api.AttributeSpec,
) ([]*api.AttributeValue, error) {
	values := s.flow.AttributeValues(name)
	matched, _, err := policy.MatchCandidateValues(attr, values, s.e.Matcher)
	return matched, err
}

func parseDefaultValue(value string) any {
	var result any
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil
	}
	return result
}

func providerSummaryFor(
	fl *api.FlowState, name api.Name,
) (policy.ProviderSummary, api.Completion) {
	return providerSummaryAt(fl, name, math.MaxInt64)
}

// providerSummaryAt summarizes the providers that had finished by sequence,
// returning the latest of their completions
func providerSummaryAt(
	fl *api.FlowState, name api.Name, seq int64,
) (policy.ProviderSummary, api.Completion) {
	deps, ok := fl.Plan.Attributes[name]
	if !ok || len(deps.Providers) == 0 {
		return policy.ProviderSummary{
			Terminal:     true,
			AllSucceeded: true,
		}, api.Completion{At: fl.CreatedAt}
	}

	res := policy.ProviderSummary{
		Terminal:     true,
		AllSucceeded: true,
	}
	var last api.Completion
	for _, sid := range deps.Providers {
		ex, ok := fl.Executions[sid]
		if !ok {
			res.Terminal = false
			res.AllSucceeded = false
			continue
		}
		terminal := policy.StepTerminal(ex.Status) && ex.Completion.Within(seq)
		if !terminal {
			res.Terminal = false
		}
		if !terminal || !policy.StepSucceeded(ex.Status) ||
			!hasValueFrom(fl, name, sid) {
			res.AllSucceeded = false
		}
		if terminal {
			last = last.Latest(ex.Completion)
		}
	}
	return res, last
}

func hasValueFrom(fl *api.FlowState, name api.Name, sid api.StepID) bool {
	for _, v := range fl.AttributeValues(name) {
		if v.Step == sid {
			return true
		}
	}
	return false
}
