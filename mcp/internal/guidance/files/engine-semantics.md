# Engine semantics

Read `/engine/api` for the full registration contract. Use `agent_capabilities` before choosing an agent provider, model, effort, or MCP tool configuration. Availability comes from the engine's workers; do not invent supported models or tools.

## Initial values and collection

`init` maps Attribute names to arrays of values. A present key declares the resolved values for that name, freezes them for this Flow, and prunes its providers even if a provider is included for other outputs. An absent key permits providers and optional fallback. `[]` explicitly declares zero values; it is not equivalent to omission or a request for a default.

Required `first` and `last` need a value and can fail with empty init. `some` and `all` accept explicitly empty init; `for_each` then creates zero work items. A required match predicate with no matching candidates can skip the Step. Optional collection falls back to its configured JSON default, or omits the input without one, when its collection policy cannot be fulfilled. An optional `some` or `all` with explicit empty init resolves to an empty array.

`preview_plan` reports selected Steps and missing external input names. It does not execute scripts, prove service availability, or guarantee runtime collection and matching will succeed. Goals name Steps, not Attributes. The `else` chain supplies fallback goal sets when an earlier set becomes impossible.

## Flow outcome and settlement

`get_flow_status` reports `active`, `completed`, or `failed`. A completed or failed Flow can still have work, child Flows, or compensation settling. Use `get_flow` and inspect `deactivated_at` to determine whether settlement has finished. `query_flows` with status `active` selects that outcome, not every Flow awaiting deactivation.

## Feeds and metrics

Each Flow, including a child Flow, reads Feed snapshots taken at its own start. Later Feed folds do not change those inputs. Feed sources and Space membership are fixed by the plan. Changing a Feed's seed, fold, Attributes, or Space starts a new version at its seed; already planned Flows continue folding into the old version.

Ordinary outputs fold when their Step completes. Outputs of compensatable Steps fold at Flow deactivation, retaining only uncompensated work. Metrics can fold earlier at Step completion, failure, or predicate/match skip; nested-flow metrics wait for all children to deactivate. A Feed watching both outputs and metrics can therefore receive separate contributions. Fold order follows transaction commits, not Step completion order. A failed fold leaves the Feed unchanged and does not fail the Flow; inspect Flow events for fold errors.
