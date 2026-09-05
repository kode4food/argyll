package policy

import (
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// WorkCompletion summarizes whether a step's work items have all reached a
// step-level outcome. It is used to decide whether to raise StepCompleted or
// StepFailed after individual work items finish
type WorkCompletion struct {
	FailureError string
	Done         bool
	Failed       bool
}

// StepParallelism returns the effective dispatch parallelism for a step. A
// missing, zero, or negative setting means one work item at a time
func StepParallelism(st *api.Step) int {
	if st.WorkConfig == nil || st.WorkConfig.Parallelism <= 0 {
		return 1
	}
	return st.WorkConfig.Parallelism
}

// CountActiveWorkItems counts currently executing work items for dispatch
// throttling
func CountActiveWorkItems(items api.WorkItems) int {
	active := 0
	for _, work := range items {
		if work.Status == api.WorkActive {
			active++
		}
	}
	return active
}

// StepWorkCompletion classifies a step's work items for step completion. One
// failure makes the step terminal at once, abandoning pending items; without
// one, the step completes only when every item has succeeded
func StepWorkCompletion(items api.WorkItems) WorkCompletion {
	res := WorkCompletion{Done: true}
	for _, work := range items {
		switch work.Status {
		case api.WorkSucceeded:
		case api.WorkFailed:
			res.Failed = true
			if res.FailureError == "" {
				res.FailureError = work.Error
			}
		case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
			res.Done = false
		}
	}
	if res.Failed {
		res.Done = true
		if res.FailureError == "" {
			res.FailureError = "work item failed"
		}
	}
	return res
}

// WorkDeadline returns when an in-flight attempt stops being expected, and
// whether it has one at all. HTTP steps use their configured timeout, which
// for async work bounds the callback; the rest fall back to the step timeout
func WorkDeadline(
	st *api.Step, work api.WorkState, fallback time.Duration,
) (time.Time, bool) {
	if st == nil || work.StartedAt.IsZero() {
		return time.Time{}, false
	}

	var ms int64
	switch {
	case WorkCompActive(work.Status):
		if st.HTTP != nil {
			ms = st.HTTP.CompensateTimeout()
		}
	case WorkActive(work.Status):
		if st.HTTP != nil {
			ms = st.HTTP.Invoke.Timeout
		}
	default:
		return time.Time{}, false
	}

	// Mirrors the client, which falls back to the engine-wide step timeout
	res := fallback
	if ms > 0 {
		res = time.Duration(ms) * time.Millisecond
	}
	if res <= 0 {
		return time.Time{}, false
	}
	return work.StartedAt.Add(res), true
}

// WorkAwaitsChildFlow reports whether an in-flight item is waiting on a child
// flow. That child's existence is the only evidence the work really launched,
// since a flow step has no timeout to expire against
func WorkAwaitsChildFlow(st *api.Step, work api.WorkState) bool {
	return st != nil && st.Type == api.StepTypeFlow && WorkActive(work.Status)
}

// WorkReadyToDispatch reports whether a step has a pending work item that can
// be claimed now, respecting parallelism and any scheduled retry time
func WorkReadyToDispatch(
	st *api.Step, ex api.ExecutionState, when time.Time,
) bool {
	if CountActiveWorkItems(ex.WorkItems) >= StepParallelism(st) {
		return false
	}

	for _, work := range ex.WorkItems {
		if !WorkPending(work.Status) {
			continue
		}
		if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
			continue
		}
		return true
	}

	return false
}
