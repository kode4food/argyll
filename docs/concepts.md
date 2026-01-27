# Concepts

This page introduces the minimal concepts you need to build apps with Argyll.

## Step

A step is a unit of work. Steps declare their inputs and outputs, and the engine decides when a step can run.

Step types:

- Sync HTTP: returns outputs in the request/response cycle
- Async HTTP: returns immediately and completes via webhook
- Script: Ale or Lua executed inside the engine
- Flow: sub-flow execution with input/output mapping

## Flow

A flow is a single execution of a plan. You start a flow by providing goals and initial inputs. The engine runs only the steps needed to satisfy the goals.

## Goals

Goals are the target steps for a flow. The engine computes the minimal set of steps required to reach those goals.

## Attributes and Arguments

- Arguments (args) are the inputs to a single step execution.
- Attributes are flow-level state produced by steps and consumed by downstream steps.

## Work Items and Tokens

Steps can expand into multiple work items (for_each). Each work item has a token. Completion events include the token so the engine can update the correct work item.

## Terminal State

A flow is terminal when it has completed or failed. A flow is deactivated once it is terminal and no active work items remain.
