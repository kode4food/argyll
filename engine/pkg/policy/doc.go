// Package policy centralizes orchestration rules that must stay consistent
// across planning, execution, recovery, and dispatch
//
// Callers provide the facts available in their layer. The planner usually has
// speculative facts such as init values and selected providers, while the
// executor has concrete flow state such as attribute values, work status, and
// provider completion. This package owns the shared decisions made from those
// facts so collect modes, status classes, and retry behavior do not drift
// between layers
package policy
