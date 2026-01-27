# Flow Design Patterns

This guide covers practical patterns for designing flows.

## Start with goals

Define clear goal steps. The engine builds the minimal execution plan needed to reach them.

## Optional inputs and defaults

Use optional attributes with defaults to keep steps reusable and avoid over-specifying initial state.

## ForEach and work items

Use `for_each` on array inputs to create parallel work items. Outputs are aggregated after all items complete.

## Predicates

Predicates let you skip work based on current attributes. Use them to enforce business rules without extra steps.

## Fail-fast vs best-effort

If a step is a goal, failure ends the flow. If it is not a goal, ensure downstream consumers handle missing outputs or use predicates to guard execution.
