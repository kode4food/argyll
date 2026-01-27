# Execution Model

Argyll is event-sourced and uses optimistic concurrency. State is derived from events, not mutated directly.

## Event-sourced state

All state changes are recorded as events. Flow state is reconstructed by applying events in order.

## Optimistic concurrency

Flow updates happen through the timebox executor. Each update runs in a transaction and retries on version conflicts.

## Deferred work execution

Step preparation (inputs, predicates, work items) happens inside the transaction. Work execution happens after commit so long-running work does not hold locks.

## Event origins

Events can be raised by:

- API calls (start flow, work completion)
- Step execution (sync steps)
- Webhook callbacks (async steps)
- Child flow completion
- Retry timers

## Tokens

Each work item has a token. Completion events include that token to update the correct work item in the flow aggregate.
