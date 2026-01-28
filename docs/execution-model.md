# Execution Model

This page explains how Argyll executes flows and why its event-sourced model delivers predictable behavior at scale. Argyll is a goal-driven orchestrator, not a process engine.

## Core idea

Argyll separates planning from execution. You declare steps and goals. The engine builds a minimal execution plan (a DAG), persists events for every state change, and executes only the work needed to satisfy the goals. There is no speculative execution, no hidden state, and no flow-level polling loop you have to manage.

## Event-sourced state

All state changes are recorded as events. Flow state is reconstructed by applying events in order, which gives you:

- A complete audit trail of what happened and why
- Deterministic recovery after crashes or restarts
- The ability to reason about correctness from the event log alone

There is no separate “source of truth” beyond the event stream.

## Optimistic concurrency (no locks or leaders)

Flow updates happen through the timebox executor. Each update runs in a transaction against the current flow state and retries on version conflicts. This enables multiple engine instances to process the same flows safely without distributed locks or leader election.

## Deferred work execution

Step preparation happens inside the transaction: inputs are collected, predicates evaluated, and work items computed. Work execution happens after commit. This keeps transactions short, avoids holding locks during long-running work, and preserves consistent, append-only event logs.

## Work items and tokens

Steps can produce multiple work items (for_each). Each work item has a token. Completion events include the token so the engine can update the correct work item and aggregate outputs for the step when all work is done.

## Event origins

Events can be raised by:

- API calls (start flow, work completion)
- Step execution (sync steps)
- Webhook callbacks (async steps)
- Child flow completion
- Retry timers

All events are recorded even if the flow is terminal. That guarantees the final state reflects all work that actually ran.

## Retries and backoff

Retries are per-work-item and configured per step. When a work item fails or reports “not completed,” the engine schedules retries using fixed, linear, or exponential backoff. The retry queue is time-based and does not require a separate scheduler process.

## Flow lifecycle

- Flow starts with goals and initial inputs
- Engine computes the minimal execution plan
- Steps execute as dependencies become satisfied
- Flow completes when all goal steps complete
- Flow fails when a goal is failed or becomes unreachable
- Flow is deactivated after it is terminal and no active work remains

This lifecycle ensures the state is explainable and aligns with the event stream.

## Why this matters (value recap)

- Deterministic execution: no hidden state, no speculative work
- Operational simplicity: scale out by adding engine instances
- Fast recovery: replay events to rebuild state
- Clear audits: every change is an event with causality
