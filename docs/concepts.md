# Concepts

This page introduces the minimal concepts you need to build apps with Argyll and explains the value you get from using it. Argyll is a goal-driven orchestrator: plan and perform only what matters.

## What Argyll is for

Argyll is a distributed, event-sourced orchestrator that executes only the work required to reach explicit goals. It is not a BPMN or human process engine. You define steps with inputs and outputs, then ask for outcomes. The engine builds the minimal execution plan, runs only what is needed, and persists every state transition as events. The value proposition is straightforward: fewer moving parts in your app, deterministic execution, and a full audit trail without writing a scheduler or custom state machine.

## Step

A step is a unit of work with declared inputs and outputs. Steps are pure declarations, and the engine decides when a step can run based on available data and goals.

Step types:

- Sync HTTP: returns outputs in the request/response cycle
- Async HTTP: returns immediately and completes via webhook
- Script: Ale or Lua executed inside the engine
- Flow: sub-flow execution with input/output mapping

## Flow

A flow is a single execution of a plan. You start a flow by providing goals and initial inputs. The engine computes a DAG of required steps, then executes them in dependency order. Because execution is goal-based, Argyll avoids speculative work and runs only what is necessary.

## Goals

Goals are the target steps for a flow. They define what “done” means. The engine walks backward from the goals to determine the minimal set of steps and inputs needed to satisfy them. Multiple goals can be declared at once; the plan is the union of their dependencies.

## Attributes and Arguments

- Arguments (args) are the inputs to a single step execution.
- Attributes are flow-level state produced by steps and consumed by downstream steps with provenance tracking.

Conceptually:

Step outputs → Flow attributes → Next step inputs

This separation keeps step execution local while the flow maintains global state in a traceable way.

## Work Items and Tokens

Steps can expand into multiple work items using for_each inputs. Each work item has a token. Completion events include the token so the engine can update the correct work item and merge outputs when the step completes.

## Predicates

Steps can include an optional predicate script (Ale or Lua) that decides whether the step should execute given its inputs. Predicates are evaluated before work starts, and a false predicate skips the step. This is a practical way to encode lightweight gating logic without adding custom branching infrastructure.

## Execution Plan (Minimal, Explicit)

Argyll builds an execution plan from goal steps and dependencies. The plan is immutable for the duration of the flow, which guarantees deterministic behavior and makes failures explainable via event history. This removes the need for custom orchestration code that mutates the plan at runtime.

## Terminal State and Deactivation

A flow is terminal when it has completed or failed. A flow is deactivated once it is terminal and no active work items remain. This separation matters: even after a flow is terminal, late work completions are still recorded for audit and compensation.

## Why this matters (value recap)

- Goal-driven execution avoids wasted work and simplifies application logic
- Event-sourced state gives a complete audit trail and deterministic recovery
- Declarative steps let you evolve flows without a custom scheduler
- Built-in async, retries, and sub-flows remove boilerplate from app code
