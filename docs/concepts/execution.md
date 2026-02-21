# Goals and Execution

Argyll is **goal-driven**: you specify what you want to achieve, and the engine automatically determines the minimal set of steps needed to accomplish it.

## Goals

A **goal** is a step that you want to complete in your flow. When you start a flow, you specify one or more goal steps. The engine uses these goals to build an execution plan.

```
Flow: "Fulfill an order"
├─ Goal: process_payment
├─ Goal: reserve_inventory
└─ Goal: send_notification
```

The engine will execute:
- All steps required by these goals
- Only those steps
- In dependency order

No speculative work. No wasted steps.

## Lazy Evaluation

The engine builds an execution plan by walking **backward** from the goal steps through their declared dependencies.

```
Step dependency graph:
  process_payment ← fetch_customer ← lookup_customer
  reserve_inventory ← check_stock ← get_inventory
  send_notification (no dependencies)

Given goals [process_payment, reserve_inventory, send_notification]:
  Required steps: fetch_customer, lookup_customer, process_payment,
                  check_stock, get_inventory, reserve_inventory,
                  send_notification

(7 steps from 30 available)
```

Only steps in the execution plan execute. This matters because it:

- **Reduces latency**: Skip unnecessary steps
- **Improves reliability**: Fewer failure points
- **Simplifies logic**: You describe what you want, not how to get it
- **Saves resources**: No background work on unnecessary computations

## Execution Plan

The execution plan is **immutable** once created. It's a DAG (directed acyclic graph) of steps computed at flow start time. This guarantees:

- **Determinism**: The plan cannot change mid-flow
- **Predictability**: You know upfront which steps will run
- **Auditability**: Every execution follows the plan

The plan is immutable by design—it removes the need for runtime orchestration logic that mutates the plan based on intermediate results.

## Multiple Goals

A flow can have multiple goals. The plan is the union of their dependencies.

```
Flow: "Process order and generate invoice"
├─ Goal: process_payment
└─ Goal: generate_invoice

Suppose generate_invoice depends on payment confirmation.
The execution plan includes both goals' dependencies.
Both complete when their goals are satisfied.
```

Goals can complete in any order. The flow completes when all goals are satisfied.

## Input Determination

The engine analyzes the execution plan to determine what inputs **must** be provided when the flow starts.

```
Step: process_payment
├─ Required inputs: customer_id, amount
├─ Optional inputs: notes
└─ Produced outputs: transaction_id, confirmation

Step: send_notification
├─ Required inputs: transaction_id (from process_payment output)
└─ Optional inputs: template

Flow analysis:
  - customer_id, amount: REQUIRED at flow start
  - notes: OPTIONAL at flow start
  - template: OPTIONAL at flow start
  - transaction_id: PRODUCED by process_payment, available to send_notification
```

The flow starts with the minimal required inputs. Optional inputs can be provided or omitted (defaults are applied only when explicitly declared on the attribute). Data produced by earlier steps automatically flows to later steps that need it.

## Why This Matters

Traditional orchestrators run all steps, or require you to manually specify which steps to run. Argyll avoids this complexity:

- **Conditional execution model**: Predicates and dependencies determine which steps run
- **No polling**: Event-driven execution, not polling for results
- **No wasted work**: Only execute what's necessary
- **Deterministic**: The plan is fixed, making failures explainable and reproducible

This is the core value proposition: **fewer moving parts in your application, more reliable orchestration**.
