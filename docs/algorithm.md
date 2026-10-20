# Execution algorithm

This is pseudocode for Argyll's execution algorithm, including planning, flow advancement, input collection, work dispatch, retries, nested flows, compensation, and recovery. It follows the implementation in `engine/internal/engine`, `engine/pkg/policy`, `engine/pkg/plan`, and `engine/pkg/step/builtins`.

## Notation and state

`EMIT` raises an event and immediately applies its reducer to the transaction's aggregate. Read `STATE` again after an emission: earlier references represent earlier versions. `AFTER_COMMIT` registers an external action for the successful attempt. An error aborts the transaction unless the pseudocode explicitly handles it. Assignments below construct local results. Aggregate changes happen through events.

```text
Flow:
    plan, init attributes, executions, metadata, tags
    status = ACTIVE | COMPLETED | FAILED
    createdAt, completedAt, deactivatedAt, error
    compensate

Step execution:
    status = PENDING | ACTIVE | COMPLETED | FAILED | SKIPPED
    startedAt, completion = { At, Seq }
    frozen inputs, retained outputs, work items, error, unsatisfied inputs

Work item:
    token, per-item inputs, outputs, startedAt, completedAt
    status = PENDING | ACTIVE | SUCCEEDED | FAILED | NOT_COMPLETED
           | COMPENSATING | COMP_PENDING | COMPENSATED | COMP_FAILED
    retryCount, nextRetryAt, error

Attribute value:
    value, producing step, SetAt
    init values have no producing step

Completion:
    At  = timestamp of the step's terminal event
    Seq = sequence of that event within its flow

TERMINAL_STEP(status): status IN {COMPLETED, FAILED, SKIPPED}
COMPLETE_STEP(status): status IN {COMPLETED, SKIPPED}
SUCCEEDED_STEP(status): status = COMPLETED
TERMINAL_FLOW(status): status IN {COMPLETED, FAILED}

WITHIN(completion, seq): completion.Seq <= seq
LATEST(a, b): b IF b.Seq > a.Seq ELSE a

VALID_WORK_TRANSITIONS:
    PENDING       -> ACTIVE | SUCCEEDED | FAILED
    ACTIVE        -> SUCCEEDED | FAILED | NOT_COMPLETED
    NOT_COMPLETED -> PENDING | SUCCEEDED | FAILED
    SUCCEEDED     -> COMPENSATING
    COMPENSATING  -> COMP_PENDING | COMPENSATED | COMP_FAILED
    COMP_PENDING  -> COMPENSATING | COMPENSATED | COMP_FAILED
    FAILED | COMPENSATED | COMP_FAILED -> none
```

Event sequences establish order within an aggregate. A transaction attempt captures one timestamp, used by its decisions and all its events. The timestamp anchors elapsed-time calculations. It is not the ordering key. Init values and the flow-start anchor use sequence zero for collection comparisons. A missing producing execution has a zero completion.

## Transactions and event application

```text
TRANSACT(command):
    REPEAT up to the configured transaction-attempt limit:
        tx.now := store.clock()
        tx.participants := empty
        tx.afterCommit := empty

        run command(tx)
            join each aggregate on first access
            load its snapshot and replay subsequent events
            retain its expected next sequence
            reuse its current transaction state on subsequent access

        atomically append events and associated store changes
            for all participating aggregates

        IF commit succeeded:
            update executor caches
            run successful attempt's AFTER_COMMIT actions
            RETURN
        IF failure is not a version conflict:
            RETURN error
        refresh or invalidate affected cached state

    RETURN retry-limit error

EMIT(type, data):
    event := { type, data, timestamp: tx.now, sequence: nextSequence }
    increment nextSequence
    STATE := REDUCE(STATE, event)
    append event to this aggregate's staged events

REDUCE(flow, event):
    flow_started:
        create ACTIVE flow and PENDING execution for each planned step
        append each init value with SetAt = event.timestamp and no producer
        set createdAt and lastUpdated
    step_started:
        freeze step inputs and create PENDING work items with assigned tokens
        set step ACTIVE and startedAt
    attribute_set:
        append { value, producing step, SetAt: event.timestamp }
    step_completed | step_failed | step_skipped:
        set terminal step status and event-specific outputs/error/inputs
        set completion = { At: event.timestamp, Seq: event.sequence }
    work_started:
        set work ACTIVE and startedAt
        clear completedAt and nextRetryAt
    work_succeeded | work_failed | work_not_completed:
        record work outcome, outputs/error, and completion timestamp
    work_retry_scheduled:
        set work PENDING, increment retryCount, record nextRetryAt
    comp_started:
        set work COMPENSATING and startedAt
        clear nextRetryAt
    comp_retry_scheduled:
        set work COMP_PENDING, increment retryCount, record nextRetryAt/error
    comp_succeeded | comp_failed:
        set work COMPENSATED or COMP_FAILED and completion/error
    flow_completed | flow_failed:
        set terminal flow status and completedAt
        set error on failure
        the completion event carries the result
        flow state derives attributes
    flow_deactivated:
        set deactivatedAt
    dispatch_deferred:
        preserve execution state
        record the deferred-dispatch fact
```

Reducers update the appropriate timestamps and payload fields. Readiness, optional defaults, and inactivity expiry are computed from the resulting state. They do not need separate flow events. WebSocket publication happens after commit and does not drive execution.

## Planning

Sources: [plan.go](../engine/pkg/plan/plan.go), [collect.go](../engine/pkg/policy/collect.go), [match.go](../engine/pkg/policy/match.go).

```text
CREATE_PLAN(request):
    load catalog
    select catalog steps in the requested space, or all steps
    require valid, nonempty goal sets and existing goal step IDs
    build attribute -> provider dependencies
    satisfied := names with nonempty init values
    available := satisfied
    satisfiableSteps := empty

    REPEAT until no progress:
        FOR each candidate step not already satisfiable:
            IF init contains a value for a required collect:NONE input:
                CONTINUE
            IF any required input other than collect:NONE is not available:
                CONTINUE
            add step to satisfiableSteps
            add its output names to available

    visitedAcrossSets := empty
    FOR each goal set, including ELSE sets:
        visited := empty
        FOR each goal in the set: COLLECT_PLAN_STEP(goal)
        add visited to visitedAcrossSets
    visited := visitedAcrossSets

    build plan.steps from included steps
    build attribute provider edges from their declared outputs
    build consumer edges for their inputs supplied by init or planned providers
    record missing external required inputs
    classify visited, excluded steps as satisfied, blocked, or missing

    FOR each included step with child goals:
        reject recursive nesting through an ancestor step
        select the child's configured catalog space
        derive placeholder init names from guaranteed parent-step inputs:
            required, const, meta, and optional inputs with defaults
        recursively build and attach the child's plan

    RETURN plan

COLLECT_PLAN_STEP(step):
    IF already visited: RETURN
    mark visited
    IF a required collect:NONE input has init values:
        record blocked inputs
        RETURN

    gate := combined required-match status using init values
        init provider state is complete only when init exists
        and no catalog provider exists for that name
        a match-evaluation error makes this planning gate UNKNOWN

    FOR each input name:
        IF gate is UNMATCHED and input has no required-match predicate:
            CONTINUE
        IF init satisfies this input:
            mark providers whose outputs are all init-satisfied as visited
            CONTINUE

        providers := catalog providers for this input
        selected := providers whose required inputs are speculatively satisfiable
        IF collect is ALL and selected excludes any provider:
            treat input as having no selected providers
        ELSE:
            mark rejected providers visited for exclusion reporting
            FOR each selected provider:
                mark provider needed
                COLLECT_PLAN_STEP(provider)

        IF input is required and has neither a selected provider nor init:
            NONE never requires an external value
            defer missing-input demand when an unresolved/closed match gate
                permits the step to be skipped without this input
            otherwise record the input as externally required

    include step if another step needs it, or its outputs are not all satisfied

INIT_SATISFIES_INPUT(input):
    require nonempty init values and collect FIRST
    without a match predicate: TRUE
    with a predicate: require the init-based match status to be MATCHED

OUTPUTS_ALREADY_SATISFIED(step):
    require at least one declared output
    require every output name to be in satisfied
```

The preview planner uses the same traversal: it selects strict providers when any exist, otherwise it includes all candidate providers for that input. A submitted plan validates its external required inputs against the actual init values before execution.

## Starting a flow

```text
START_FLOW(request):
    validate request
    plan := CREATE_PLAN(request)
    START_PLAN(request.id, plan, request.init, request.tags,
               request.compensate)

START_PLAN(id, plan, options):
    TRANSACT:
        join flow id
        validate parent metadata and external required init inputs
        IF flow already exists:
            IF original start matches id, goal sets, and init: RETURN
            RETURN flow-already-exists error
        EMIT flow_started(plan, init, metadata, tags, compensate)
        ADVANCE_FLOW()
```

## Collect policy and match gates

Values retain their persisted arrival order. Match predicates filter candidate values without changing that order, whether the input is required or optional. A match error propagates to step preparation, which records step failure. An unmatched required input can close the step entirely. An unmatched optional input never does: it falls back to its default through the same path as any other unfulfilled optional input, since the step does not depend on it.

A `for_each` input is valid only with `collect: all` or `collect: some`: its resolved value is the collected candidate list itself, which step preparation fans out into one work item per element. An empty filtered candidate list is unambiguous absence, letting `INPUT_FULFILLED` and the optional default apply normally.

```text
INPUT_VALUES(name, input):
    values := STATE.attributes[name]
    IF input has a match predicate:
        RETURN values for which the predicate returns TRUE
    RETURN values

PROVIDER_SUMMARY(name, cutoff = INFINITY):
    IF name has no planned providers:
        RETURN { terminal: TRUE, allSucceeded: TRUE },
               Completion(STATE.createdAt, 0)

    summary := { terminal: TRUE, allSucceeded: TRUE }
    last := zero Completion
    FOR each planned provider:
        execution := STATE.executions[provider]
        terminal := execution exists AND TERMINAL_STEP(execution.status)
                    AND WITHIN(execution.completion, cutoff)
        IF NOT terminal: summary.terminal := FALSE
        IF NOT terminal OR execution.status != COMPLETED
           OR no recorded attribute value came from this provider:
            summary.allSucceeded := FALSE
        IF terminal: last := LATEST(last, execution.completion)
    RETURN summary, last

INPUT_FULFILLED(mode, values, providers):
    FIRST: RETURN count(values) > 0
    LAST:  RETURN count(values) > 0 AND providers.terminal
    SOME:  RETURN count(values) > 0 AND providers.terminal
    ALL:   RETURN count(values) > 0 AND providers.terminal
                   AND providers.allSucceeded
    NONE:  RETURN count(values) = 0 AND providers.terminal

RESOLVE_VALUE(mode, values):
    IF values is empty: RETURN OMIT
    FIRST: RETURN first value
    LAST:  RETURN last value
    SOME | ALL: RETURN array of values in persisted order
    NONE: RETURN OMIT

MATCH_STATUS(input, candidateValues, providers):
    IF no match predicate: RETURN NOT_GATED
    matched, unmatched := evaluate predicate against all candidateValues
    NONE:
        IF matched is nonempty: RETURN UNMATCHED
        IF providers.terminal: RETURN MATCHED
    LAST:
        IF providers.terminal:
            RETURN MATCHED if matched is nonempty ELSE UNMATCHED
    ALL:
        IF unmatched is nonempty: RETURN UNMATCHED
        IF providers.terminal:
            RETURN MATCHED if matched is nonempty AND providers.allSucceeded
                   ELSE UNMATCHED
    FIRST | SOME:
        IF matched is nonempty: RETURN MATCHED
        IF providers.terminal: RETURN UNMATCHED
    RETURN UNKNOWN

CLOSED_MATCH_INPUTS(step):
    result := empty
    FOR each required input with a match predicate:
        providers := PROVIDER_SUMMARY(input.name)
        IF NOT providers.terminal: CONTINUE
        IF MATCH_STATUS(input, all recorded values, providers) = UNMATCHED:
            append input.name to result
    RETURN sorted result
```

Combined planning match status is UNMATCHED if any gate is unmatched, otherwise UNKNOWN if any is unknown, otherwise MATCHED if any is matched, otherwise NOT_GATED. Runtime closed-match checks wait for terminal providers before using a match gate to skip the step.

## Required readiness and optional decisions

Sources: [step-start.go](../engine/internal/engine/step-start.go), [collect.go](../engine/pkg/policy/collect.go).

```text
REQUIRED_READY_AT(step):
    anchor := Completion(STATE.createdAt, 0)
    FOR each required input:
        values := INPUT_VALUES(input.name, input)
        providers, last := PROVIDER_SUMMARY(input.name)
        IF NOT INPUT_FULFILLED(input.collect, values, providers):
            RETURN NOT_READY
        IF input.collect = FIRST:
            last := completion of the first matched value's producing step
                or zero Completion for an init value
        anchor := LATEST(anchor, last)
    RETURN anchor

COLLECT_WINDOW(anchorAt, interval, values):
    deadline := anchorAt + interval
    accepted := empty
    FOR each value in persisted order:
        IF value.SetAt >= deadline: BREAK
        append value to accepted
        deadline := MAX(deadline, value.SetAt + interval)
    RETURN accepted, deadline

FULFILLED_AT(mode, accepted, latestProviderAt):
    FIRST: RETURN first accepted value's SetAt
    NONE:  RETURN latestProviderAt
    LAST:  RETURN MAX(last accepted value's SetAt, latestProviderAt)
    SOME | ALL:
        RETURN MAX(all accepted SetAt timestamps, latestProviderAt)

OPTIONAL_DECISION(step, input, anchor, now):
    IF anchor is NOT_READY: RETURN WAIT_WITHOUT_TIMER
    values := INPUT_VALUES(input.name, input)
    IF obtaining values failed: RETURN WAIT_WITHOUT_TIMER

    IF input.deadline = 0:
        accepted := values whose producing execution's completion.Seq
                    is <= anchor.Seq
        providers := PROVIDER_SUMMARY(input.name, anchor.Seq)
        IF INPUT_FULFILLED(input.collect, accepted, providers):
            RETURN USE(accepted)
        RETURN FALLBACK

    accepted, deadline := COLLECT_WINDOW(anchor.At, input.deadline, values)
    providers, last := PROVIDER_SUMMARY(input.name)
    IF INPUT_FULFILLED(input.collect, accepted, providers):
        IF FULFILLED_AT(input.collect, accepted, last.At) < deadline:
            RETURN USE(accepted)
    IF providers.terminal OR now >= deadline:
        RETURN FALLBACK
    IF input.collect = NONE AND accepted is nonempty:
        RETURN FALLBACK
    IF input.collect = ALL AND NOT CAN_COLLECT_ALL(input.name):
        RETURN FALLBACK
    RETURN WAIT_UNTIL(deadline)

OPTIONAL_DECIDED(step, input, now):
    anchor := REQUIRED_READY_AT(step)
    IF computing anchor failed: RETURN FALSE
    RETURN OPTIONAL_DECISION(step, input, anchor, now) is USE or FALLBACK
```

For Deadline 0, init values qualify at sequence zero. Provider completion, failure, or skipping after the required anchor cannot establish collection satisfaction at that anchor. In particular, a subsequent unneeded-provider skip cannot turn `collect: none` from FALLBACK into confirmed absence. For a positive interval, only values strictly before expiry extend the window. Provider completion without a value does not extend it. Fallback uses the configured JSON default, or omits the input when no default exists. Neither timeout nor impossible collection uses partial results.

## Goals, reachability, and output demand

Source: [flow-state.go](../engine/internal/engine/flow-state.go).

```text
GOAL_SET_FAILED(goals):
    IF any goal step is FAILED: RETURN TRUE
    RETURN every goal was pruned by a required-match skip

ACTIVE_GOALS():
    RETURN first goal set in the ordered ELSE chain that has not failed
           or NONE if every set failed

START_GOALS():
    RETURN ACTIVE_GOALS().steps, or empty

KEPT_GOALS():
    active := ACTIVE_GOALS()
    IF active is NONE: RETURN all goals from all sets
    IF GOALS_COMPLETE(active.steps): RETURN active.steps
    RETURN all goals in active and its remaining ELSE sets

GOAL_SCOPE(goals):
    walk backwards from goals through every input's planned providers
    RETURN all reachable step IDs, including goals

CAN_COLLECT_ALL(name):
    IF name has no plan edges: RETURN FALSE
    FOR each terminal provider:
        IF it did not complete successfully or has no recorded value:
            RETURN FALSE
    RETURN TRUE

HAS_INPUT_PROVIDER(name):
    IF name has no plan edges: RETURN FALSE
    IF edges have no providers: RETURN TRUE
    FOR each provider:
        IF provider is terminal:
            IF COMPLETED and its retained outputs contain name: RETURN TRUE
        ELSE IF CAN_STEP_COMPLETE(provider):
            RETURN TRUE
    RETURN FALSE

CAN_STEP_COMPLETE(step):
    IF plan excludes step as already satisfied: RETURN TRUE
    IF step is terminal: RETURN step.status = COMPLETED
    IF computing closed-match inputs fails: RETURN FALSE
    IF closed-match inputs are nonempty: RETURN TRUE
        // Preparation can intentionally skip this step
    IF any required-match input still has nonterminal providers: RETURN TRUE

    FOR each required input:
        hasValue := HAS_INPUT_VALUE(input.name, input)
        NONE: possible := NOT hasValue
        ALL:  possible := CAN_COLLECT_ALL(input.name)
                          AND (hasValue OR HAS_INPUT_PROVIDER(input.name))
        FIRST | LAST | SOME:
              possible := hasValue OR HAS_INPUT_PROVIDER(input.name)
        IF NOT possible: RETURN FALSE
    RETURN TRUE

GOAL_SET_IMPOSSIBLE(goals):
    RETURN GOAL_SET_FAILED(goals)
        OR any goal not pruned by required match cannot complete

FLOW_FAILED():
    RETURN every goal set is impossible

HAS_INPUT_VALUE(name, input):
    values, error := INPUT_VALUES(name, input)
    RETURN no error AND values is nonempty

NEEDS_OUTPUT(name, scope, now):
    FOR each planned consumer of name:
        IF consumer is not PENDING or not in scope: CONTINUE
        input := consumer's attribute named name
        IF input is absent: CONTINUE
        IF closed-match evaluation succeeds and finds closed inputs: CONTINUE
        IF input is optional AND OPTIONAL_DECIDED(consumer, input, now):
            CONTINUE
        hasValue := HAS_INPUT_VALUE(name, input)
        IF input.collect = ALL AND NOT CAN_COLLECT_ALL(name): CONTINUE
        IF NOT hasValue OR input.collect != FIRST: RETURN TRUE
    RETURN FALSE

OUTPUTS_NEEDED(step, goals, now):
    IF step is itself in goals: RETURN TRUE
    scope := GOAL_SCOPE(goals)
    RETURN any declared output satisfies NEEDS_OUTPUT(name, scope, now)
```

Output demand is shared across consumers: one decided optional does not remove demand from another pending consumer, including a required consumer. Only PENDING executions are eligible for the unused-provider skip.

## Advancing a flow

Source: [flow-continue.go](../engine/internal/engine/flow-continue.go).

```text
ADVANCE_FLOW():
    SETTLE_PENDING()
    CHECK_TERMINAL()
    START_READY_PENDING_STEPS()
    SCHEDULE_OPTIONAL_TIMEOUTS()

SETTLE_PENDING():
    LOOP:
        FOR each PENDING step:
            IF NOT CAN_STEP_COMPLETE(step):
                collect its currently derivable inputs
                EMIT step_failed(required-input-unavailable or input error)
                RESTART LOOP

        FOR each PENDING step:
            IF NOT OUTPUTS_NEEDED(step, KEPT_GOALS(), tx.now):
                EMIT step_skipped(reason = "outputs not needed")
                RESTART LOOP
        RETURN

CAN_START(step, now):
    IF step is not PENDING: RETURN FALSE
    IF NOT OUTPUTS_NEEDED(step, START_GOALS(), now): RETURN FALSE
    IF closed-match inputs are nonempty OR match evaluation failed:
        RETURN TRUE
        // Preparation records the skip or failure
    anchor := REQUIRED_READY_AT(step)
    IF computing anchor failed: RETURN TRUE
    IF anchor is NOT_READY: RETURN FALSE
    RETURN every optional input's decision is USE or FALLBACK

START_READY_PENDING_STEPS():
    WHILE flow is not terminal:
        ready := PENDING step IDs satisfying CAN_START(step, tx.now)
                 sorted by step ID
        IF ready is empty: RETURN
        FOR each step in ready:
            read current STATE
            IF flow is terminal: RETURN
            IF step is no longer PENDING: CONTINUE
            PREPARE_STEP(step)
```

Unreachable-step failures are exhausted before each unused-step skip. Inline completion and intentional skipping call `ADVANCE_FLOW` again inside the same transaction, using fresh aggregate state.

## Preparing a step and freezing inputs

```text
COLLECT_STEP_INPUTS(step):
    anchor := REQUIRED_READY_AT(step)
    result := empty arguments
    FOR each attribute:
        IF const:
            result[name] := parsed JSON constant, or null if invalid JSON
            CONTINUE
        IF not an input: CONTINUE

        IF optional:
            decision := OPTIONAL_DECISION(step, input, anchor, tx.now)
            IF decision is waiting: CONTINUE
            IF decision = FALLBACK:
                IF configured default exists:
                    value := parsed JSON default, or null if invalid JSON
                    map value and store under the handler argument name
                CONTINUE
            values := decision.values
        ELSE:
            values := INPUT_VALUES(name, input)

        value := RESOLVE_VALUE(input.collect, values)
        IF value is not OMIT:
            apply input mapping
            on mapping failure retain original value
            store under the handler argument name
    RETURN result

PREPARE_STEP(step):
    require current execution is PENDING
    closed := CLOSED_MATCH_INPUTS(step)
    IF match evaluation failed:
        EMIT step_failed(predicate error)
        ADVANCE_FLOW()
        RETURN

    inputs := COLLECT_STEP_INPUTS(step)
    IF collection failed:
        EMIT step_failed(input/predicate error)
        ADVANCE_FLOW()
        RETURN
    IF closed is nonempty:
        EMIT step_skipped(required-match reason, inputs, closed)
        ADVANCE_FLOW()
        RETURN
    items := COMPUTE_WORK_ITEMS(step, inputs)
    FOR each item:
        evaluate step predicate on inputs overlaid with item inputs
        IF predicate errors:
            EMIT step_failed(predicate error, inputs)
            ADVANCE_FLOW()
            RETURN
        IF predicate is FALSE: exclude item
        ELSE: assign a fresh receipt token
    IF items had candidates and every candidate was excluded:
        EMIT step_skipped(predicate-false reason, inputs)
        ADVANCE_FLOW()
        RETURN
    EMIT step_started(inputs, items)
    CONTINUE_STEP_WORK(step)

COMPUTE_WORK_ITEMS(step, inputs):
    arrays := array-valued inputs marked for_each, under mapped argument names
    ignore values that cannot be converted to arrays
    IF arrays is empty: RETURN one item with no per-item overrides
    validate Cartesian product size does not overflow
    IF any selected array is empty: RETURN no items
    RETURN Cartesian product of arrays as per-item input overrides
```

The predicate is evaluated once per candidate Work Item after expansion. Without `for_each`, the single candidate receives the collected inputs unchanged. With `for_each`, each candidate receives its elements overlaid on those inputs. If every candidate is rejected, the Step is skipped. An empty work-item product has no candidates to evaluate and completes without work. `step_started` freezes the consumer's inputs and accepted work items. Dispatch, retries, and recovery do not re-evaluate predicates.

## Work dispatch and invocation

Sources: [work-start.go](../engine/internal/engine/work-start.go), [step-dispatch.go](../engine/internal/engine/step-dispatch.go), [builtins](../engine/pkg/step/builtins).

```text
CONTINUE_STEP_WORK(step):
    started := START_PENDING_WORK(step)
    IF started is nonempty:
        IF step is not a nested flow:
            AFTER_COMMIT invoke started non-inline work items concurrently
    ELSE IF execution has no work items:
        SETTLE_STEP(step)

START_PENDING_WORK(step):
    require step ACTIVE
    limit := configured parallelism, default 1
    canDispatch := local health is unknown or is not UNHEALTHY
    started := empty

    FOR each work token in sorted order:
        read current execution
        IF execution is no longer ACTIVE or active item count >= limit: BREAK
        IF item is not PENDING or its nextRetryAt > tx.now: CONTINUE
        inputs := frozen step inputs overlaid with item inputs
        IF handling is memoized and cache contains outputs for step + inputs:
            EMIT work_succeeded(cached outputs)
            SETTLE_STEP(step)
            RETURN started
        IF NOT canDispatch: CONTINUE

        START_WORK_ITEM(step, token, inputs)
        add current work item to started

    IF local dispatch is unavailable and due pending work has capacity:
        EMIT dispatch_deferred
        schedule keyed dispatch wakeup after local dispatch backoff
    RETURN started

START_WORK_ITEM(step, token, inputs):
    validate work transition to ACTIVE
    EMIT work_started(token, inputs)
    IF handler is inline:
        invoke handler inside this transaction
        handler completion/failure joins this transaction
        RETURN
    schedule attempt deadline if applicable
    IF step is a nested flow:
        START_CHILD_FLOW(step, token, inputs)

INVOKE_WORK_ITEM(step, token):
    inputs := frozen step inputs overlaid with work-item inputs
    invoke registered handler with flow/step IDs, token, and metadata
    IF handler returns an error: SETTLE_HANDLER_ERROR(error)

HTTP_INVOKE(step, token, inputs):
    construct metadata, receipt token, and callback URL for async invocation
    inject declared meta attributes into handler arguments
    resolve endpoint placeholders
    encode arguments and metadata
    send configured HTTP method with configured/default request timeout
    transport/read failures and HTTP 5xx => retryable NOT_COMPLETED error
    other non-2xx responses => permanent failure
    decode successful response as output arguments
    IF asynchronous: wait for callback or attempt deadline
    ELSE: COMPLETE_WORK(outputs)

SCRIPT_INVOKE(step, token, inputs):
    compile script, inject meta arguments, evaluate inline
    report compilation/evaluation failure and update handler health
    on success COMPLETE_WORK(outputs) and mark handler healthy

SETTLE_HANDLER_ERROR(error):
    IF invalid work transition: ignore stale/duplicate settlement
    ELSE IF error is retryable NOT_COMPLETED: NOT_COMPLETE_WORK(error)
    ELSE: FAIL_WORK(error)

ON_DISPATCH_WAKEUP(step):
    IF flow missing, terminal, deactivated, or step not ACTIVE: RETURN
    IF due pending work has capacity but local health prevents dispatch:
        ask scheduler to retry this delivery
    ELSE:
        start eligible pending work and invoke it after commit
```

The HTTP callback validates the flow, execution, receipt token, and action. Invocation callbacks route output objects to `COMPLETE_WORK`, and problem responses to `FAIL_WORK`. Stale invocation transitions are acknowledged as duplicates. Compensation callbacks similarly settle only the compensation phase.

## Work settlement and retained outputs

Sources: [work-stop.go](../engine/internal/engine/work-stop.go), [step-stop.go](../engine/internal/engine/step-stop.go), [work-items.go](../engine/internal/engine/work-items.go).

```text
COMPLETE_WORK(step, token, outputs):
    TRANSACT or reuse the current inline/parent-child transaction:
        validate work transition to SUCCEEDED
        for memoized handling, attempt to cache raw outputs for item inputs
        EMIT work_succeeded(token, raw outputs)
        SETTLE_STEP(step)

FAIL_WORK(step, token, error):
    validate work transition to FAILED
    EMIT work_failed(token, error)
    SETTLE_STEP(step)

NOT_COMPLETE_WORK(step, token, error):
    validate work transition to NOT_COMPLETED
    EMIT work_not_completed(token, error)
    IF flow is terminal:
        MAYBE_DEACTIVATE()
        RETURN
    IF step is not ACTIVE: RETURN
    SCHEDULE_RETRY(step, token)
    CONTINUE_STEP_WORK(step)
    SETTLE_STEP(step)

SETTLE_STEP(step):
    settled := FALSE
    IF step is ACTIVE:
        settled := CHECK_STEP_COMPLETION(step) settled the step
    read current STATE
    IF step is FAILED:
        start compensation for any newly succeeded work items if supported
    IF flow is terminal:
        MAYBE_DEACTIVATE()
        RETURN
    IF settled:
        ADVANCE_FLOW()
    ELSE IF step is ACTIVE:
        CONTINUE_STEP_WORK(step)

CHECK_STEP_COMPLETION(step):
    require step ACTIVE
    IF any work item is FAILED:
        EMIT step_failed(first available failure reason, or generic failure)
        RETURN SETTLED
    IF any item is PENDING, ACTIVE, or NOT_COMPLETED:
        RETURN UNSETTLED

    outputs := COLLECT_STEP_OUTPUTS(step)
    IF step is not itself a kept goal:
        retain only declared outputs for which
            NEEDS_OUTPUT(name, GOAL_SCOPE(KEPT_GOALS()), tx.now) is TRUE
    FOR each retained declared output:
        EMIT attribute_set(step, name, value)
    EMIT step_completed(retained outputs,
                        duration = MAX(0, tx.now - step.startedAt))
    RETURN SETTLED

COLLECT_STEP_OUTPUTS(step):
    succeeded := work items with status SUCCEEDED
    IF count = 0: RETURN no outputs
    IF count = 1: raw := that item's outputs
    ELSE:
        FOR each output of each succeeded item:
            append an object containing that output and the item's
                for_each argument values to an array for the output name
        raw := resulting arrays by output name
    map declared outputs from raw using output mappings or handler names
    omit absent outputs and failed output mappings
    RETURN mapped outputs
```

Raw work outputs remain recorded by `work_succeeded`. The filtered output set becomes both the step's outputs and its `attribute_set` events. A value needed only by decided optional consumers can therefore remain in work history without becoming a flow attribute. Goal-step outputs are retained. Late outcomes can settle already-running work after flow failure, but cannot restart terminal steps or change frozen inputs.

## Work retries and attempt deadlines

```text
RESOLVE_RETRY_CONFIG(step):
    start with engine defaults
    override MaxRetries when the step value is nonzero
    override positive initial/maximum backoff and nonempty backoff type

SHOULD_RETRY(work, config):
    IF config.MaxRetries < 0: RETURN TRUE
    RETURN config.MaxRetries > 0 AND work.retryCount < config.MaxRetries

RETRY_AT(work, config):
    fixed:       delay := config.InitBackoff
    linear:      delay := config.InitBackoff * (work.retryCount + 1)
    exponential: delay := config.InitBackoff * 2 ^ work.retryCount
    unknown backoff type: use fixed
    delay := MIN(delay, config.MaxBackoff)
    RETURN tx.now + delay

SCHEDULE_RETRY(step, token):
    IF step is not ACTIVE or item is not NOT_COMPLETED: RETURN
    config := RESOLVE_RETRY_CONFIG(step)
    IF SHOULD_RETRY(item, config):
        at := RETRY_AT(item, config)
        EMIT work_retry_scheduled(token, incremented retryCount, at)
        schedule keyed retry task at at
    ELSE:
        EMIT work_failed(token, last error)

ON_RETRY_WAKEUP(step, token):
    IF flow missing/terminal, step not ACTIVE, or item missing: RETURN
    IF item is claimable but local dispatch unavailable:
        ask scheduler to retry delivery
    IF item is not PENDING: RETURN
    IF nextRetryAt > tx.now:
        re-arm retry at nextRetryAt
        RETURN
    IF parallelism is exhausted: RETURN
    START_WORK_ITEM(step, token, frozen step inputs overlaid with item inputs)
    AFTER_COMMIT execute the started work through its handler

WORK_DEADLINE(step, item):
    IF nested-flow step or item has no startedAt: RETURN NONE
    IF item is ACTIVE:
        timeout := invoke timeout, otherwise engine default
    ELSE IF item is COMPENSATING:
        timeout := compensation timeout, otherwise engine default
    ELSE: RETURN NONE
    IF timeout <= 0: RETURN NONE
    RETURN item.startedAt + timeout

ON_WORK_DEADLINE(step, token):
    deadline := WORK_DEADLINE(current step, current item)
    IF no deadline: RETURN
    IF deadline > tx.now:
        re-arm at deadline
        RETURN
    IF item is COMPENSATING:
        REQUEST_COMPENSATION_RETRY(deadline-exceeded error)
    ELSE:
        NOT_COMPLETE_WORK(deadline-exceeded error)
```

## Optional timers and scheduled delivery

```text
SCHEDULE_OPTIONAL_TIMEOUTS():
    IF flow is terminal: RETURN
    FOR each PENDING step with positive optional deadlines:
        anchor := REQUIRED_READY_AT(step)
        IF anchor unavailable or evaluation failed: CONTINUE
        FOR each optional input with deadline > 0:
            decision := OPTIONAL_DECISION(step, input, anchor, tx.now)
            IF decision = WAIT_UNTIL(at):
                schedule keyed task (flow, timeout, step, input) at at

ON_OPTIONAL_TIMEOUT(step):
    IF flow terminal, step missing, or step not PENDING: RETURN
    ADVANCE_FLOW()

SCHEDULE(key, at, message):
    join Timebox scheduler state in the current transaction
    create or replace the durable task if deadline/message changed
    leave an identical existing task unchanged

ON_DUE_TASK(task):
    Timebox consumes the task and runs its processor in one transaction
    join the target flow in that transaction
    route message to optional timeout, work deadline, retry,
        compensation retry, dispatch, or flow reconciliation
    IF dispatch unavailable: roll back delivery and request scheduler retry
    otherwise commit consumption and any events or schedules produced
```

Tasks are not cancelled when their original purpose disappears. Delivery rechecks current state. An optional timeout is a wakeup to apply policy, not a separate expiry event. If an early consumption finds an open window, normal advancement re-arms it in that same transaction.

## Nested flows

```text
START_CHILD_FLOW(step, token, workInputs):
    childID := deterministic ID from parent flow ID, step ID, and work token
    childInit := each work input wrapped as one init value
    childMetadata := inherited metadata plus parent flow/step/token
    childCompensate := step's configured child-flow compensation option
    join child aggregate in current transaction
    START_PLAN(childID, precomputed child plan, child options)

COMPLETE_PARENT_WORK(child):
    IF child has no parent metadata: RETURN no parent
    join parent aggregate in current transaction
    require parent exists
    item := referenced parent work item
    IF item no longer accepts a result: RETURN current parent
    IF child COMPLETED:
        outputs := child's first recorded value for each attribute
        validate presence of every mapped declared parent-step output
        IF validation fails: FAIL_WORK(parent item, output error)
        ELSE: COMPLETE_WORK(parent item, outputs)
    ELSE:
        FAIL_WORK(parent item, child error or generic child failure)
    RETURN current parent
```

Parent work start and child creation commit together. Child outcome and parent work settlement also commit together. Nested-flow attempts do not use the ordinary work deadline. Their child flows control completion.

## Flow completion and deactivation

```text
GOALS_COMPLETE(goals):
    RETURN each goal is excluded as satisfied, COMPLETED, or SKIPPED

FLOW_COMPLETE():
    active := ACTIVE_GOALS()
    RETURN active exists AND GOALS_COMPLETE(active.steps)
        AND every planned step is terminal

FAIL_ABANDONED_WORK():
    FOR each execution whose step or flow is terminal:
        FOR each PENDING work item:
            EMIT work_failed("execution no longer active")

CHECK_TERMINAL():
    FAIL_ABANDONED_WORK()
    IF flow already terminal: RETURN
    IF FLOW_COMPLETE():
        result := merge winning goal executions' retained outputs
        add first attribute values for goal outputs excluded as satisfied
        EMIT flow_completed(result)
        MAYBE_DEACTIVATE()
    ELSE IF FLOW_FAILED():
        EMIT flow_failed(a failed step's reason, or generic flow failure)
        MAYBE_DEACTIVATE()

MAYBE_DEACTIVATE():
    IF flow not terminal or already deactivated: RETURN
    FAIL_ABANDONED_WORK()
    COMPENSATE_FLOW()
    parent := COMPLETE_PARENT_WORK(STATE)
    IF any work item is PENDING, ACTIVE, COMP_PENDING, or COMPENSATING: RETURN
    IF succeeded work still requires supported compensation: RETURN
    IF parent has not released this child: RETURN
    EMIT flow_deactivated(current terminal status)
    FOR each child identified by a nested-flow work token:
        join child in the same transaction
        MAYBE_DEACTIVATE(child)

PARENT_RELEASED(child, parent):
    RETURN child has no parent
        OR parent is absent/deactivated
        OR referenced parent work is FAILED, COMPENSATED, or COMP_FAILED
```

A terminal flow has an outcome. A deactivated flow also has no outstanding work, compensation, or parent retention. Successful child flows can remain retained while the parent may still request rollback.

## Compensation

Source: [work-compensate.go](../engine/internal/engine/work-compensate.go).

```text
FLOW_ROLLS_BACK(step):
    IF flow compensation disabled or flow not terminal: RETURN FALSE
    IF flow FAILED: RETURN TRUE
    RETURN step is outside the winning goal set's dependency scope

COMPENSATE_FLOW():
    IF compensation disabled or flow not terminal: RETURN
    IF any compensation item is COMP_PENDING or COMPENSATING: RETURN
    pending := steps with SUCCEEDED work, a compensator,
               and FLOW_ROLLS_BACK(step)
    wave := pending steps with no downstream pending step,
            including downstream steps reached transitively
    FOR each step in wave: START_STEP_COMPENSATIONS(step)

START_STEP_COMPENSATIONS(step):
    IF no compensator: RETURN
    FOR each SUCCEEDED work item:
        args := attributes marked compensated
            input attributes read from frozen inputs overlaid with item inputs
            output attributes read from the item's raw outputs
            use the handler's mapped argument names
        validate transition and EMIT comp_started(token)
        schedule compensation attempt deadline when applicable
        AFTER_COMMIT invoke compensator(token, args)

ON_COMPENSATION_SUCCESS(step, token):
    IF item not COMP_PENDING or COMPENSATING: RETURN
    EMIT comp_succeeded(token)
    MAYBE_DEACTIVATE()

ON_COMPENSATION_FAILURE(step, token, permanentError):
    IF item not COMP_PENDING or COMPENSATING: RETURN
    EMIT comp_failed(token, permanentError)
    MAYBE_DEACTIVATE()

REQUEST_COMPENSATION_RETRY(step, token, retryableError):
    IF item not COMPENSATING: RETURN
    IF SHOULD_RETRY(item, RESOLVE_RETRY_CONFIG(step)):
        at := RETRY_AT(item, config)
        EMIT comp_retry_scheduled(token, incremented retryCount, at, error)
        schedule keyed compensation retry at at
    ELSE:
        ON_COMPENSATION_FAILURE(step, token, error)

ON_COMPENSATION_RETRY(step, token):
    IF flow missing, item missing, or item not COMP_PENDING: RETURN
    IF nextRetryAt > tx.now:
        re-arm at nextRetryAt
        RETURN
    IF local dispatch unavailable:
        request scheduler retry
        RETURN
    EMIT comp_started(token)
    schedule compensation attempt deadline when applicable
    AFTER_COMMIT invoke compensator using reconstructed compensation arguments
```

Step failure also compensates that step's already-succeeded work when a compensator exists, independently of whole-flow rollback. Flow rollback proceeds from downstream consumers toward upstream providers, one settled wave at a time. Ordinary work and compensation use the item's recorded retry counter and the same retry configuration.

## Startup, replay, and recovery

Sources: [engine-start.go](../engine/internal/engine/engine-start.go), [recover.go](../engine/internal/engine/recover.go), [reconcile.go](../engine/internal/engine/reconcile.go).

```text
START_ENGINE():
    load this node's health state
    start durable scheduler processing
    list flows in the active store index, including terminal retained flows
    schedule immediate reconciliation for each indexed flow

RECOVER_FLOW(flowID):
    TRANSACT:
        join flow
        restore snapshot and replay remaining events in sequence
        IF flow absent or deactivated: RETURN
        validate parent metadata

        FOR each step with a compensator:
            re-arm COMP_PENDING items at MAX(tx.now, nextRetryAt)
            IF succeeded items need whole-flow rollback:
                COMPENSATE_FLOW()
            ELSE IF succeeded items need compensation because the step failed:
                SETTLE_STEP(step)

        FOR every work item:
            IF WORK_DEADLINE(step, item) exists:
                restore its attempt-deadline task

        FOR each ACTIVE step with no work items:
            CONTINUE_STEP_WORK(step)

        IF flow terminal:
            MAYBE_DEACTIVATE()
            RETURN

        FOR each ACTIVE step with due PENDING work and parallelism capacity:
            schedule immediate work-dispatch task
        FOR each PENDING work item:
            IF nextRetryAt exists: restore retry task at nextRetryAt
            ELSE IF execution ACTIVE: schedule immediate retry task
        ADVANCE_FLOW()

ON_COMMITTED_DISPATCH_DEFERRED(flowID):
    schedule flow reconciliation after local dispatch backoff

STOP_ENGINE():
    cancel engine context
    wait for scheduler to stop
    attempt to save catalog and cluster snapshots
    close storage backend
```

Replay reconstructs attribute arrival history, frozen execution inputs, work state, and terminal step completions. Recovery derives required anchors and optional windows from those facts, restores outstanding wakeups, and resumes the same advancement algorithm. It does not rebuild a separate optional-decision projection or invent decision events.
