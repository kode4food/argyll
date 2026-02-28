package engine

import (
	"fmt"

	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type healthResolver struct {
	catState *api.CatalogState
	steps    api.Steps
	base     map[api.StepID]*api.HealthState
	cache    map[api.StepID]*api.HealthState
	visiting map[api.StepID]bool
	plans    map[api.StepID]*api.ExecutionPlan
	planErrs map[api.StepID]error
}

// UpdateStepHealth updates the health status of a registered step, used
// primarily for tracking HTTP service availability and script errors
func (e *Engine) UpdateStepHealth(
	stepID api.StepID, health api.HealthStatus, errMsg string,
) error {
	cmd := func(st *api.PartitionState, ag *PartitionAggregator) error {
		if stepHealth, ok := st.Health[stepID]; ok {
			if stepHealth.Status == health && stepHealth.Error == errMsg {
				return nil
			}
		}

		return events.Raise(ag, api.EventTypeStepHealthChanged,
			api.StepHealthChangedEvent{
				StepID: stepID,
				Status: health,
				Error:  errMsg,
			},
		)
	}

	_, err := e.execPartition(cmd)
	return err
}

// ResolveHealth returns resolved health for all steps, deriving flow step
// health from all steps included in the flow's execution preview
func ResolveHealth(
	catState *api.CatalogState, base map[api.StepID]*api.HealthState,
) map[api.StepID]*api.HealthState {
	if catState == nil {
		return map[api.StepID]*api.HealthState{}
	}

	resolver := &healthResolver{
		catState: catState,
		steps:    catState.Steps,
		base:     base,
		cache:    map[api.StepID]*api.HealthState{},
		visiting: map[api.StepID]bool{},
		plans:    map[api.StepID]*api.ExecutionPlan{},
		planErrs: map[api.StepID]error{},
	}

	resolved := make(map[api.StepID]*api.HealthState, len(catState.Steps))
	for stepID := range catState.Steps {
		resolved[stepID] = resolver.resolve(stepID)
	}
	return resolved
}

func (r *healthResolver) resolve(stepID api.StepID) *api.HealthState {
	if h, ok := r.cache[stepID]; ok {
		return h
	}

	if base, ok := r.base[stepID]; ok && base != nil {
		if base.Status != api.HealthUnknown {
			r.cache[stepID] = base
			return base
		}
	}

	step, ok := r.steps[stepID]
	if !ok {
		h := &api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step not found: %s", stepID),
		}
		r.cache[stepID] = h
		return h
	}

	if r.visiting[stepID] {
		h := &api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow health cycle at step %s", stepID),
		}
		r.cache[stepID] = h
		return h
	}

	if step.Type != api.StepTypeFlow || step.Flow == nil {
		h := baseHealth(stepID, r.base)
		r.cache[stepID] = h
		return h
	}

	r.visiting[stepID] = true
	defer delete(r.visiting, stepID)

	plan, err := r.previewFlowPlan(stepID, step)
	if err != nil {
		h := &api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow preview failed for %s: %v", stepID, err),
		}
		r.cache[stepID] = h
		return h
	}

	var unknown *api.HealthState
	for plannedStepID := range plan.Steps {
		plannedStepHealth := r.resolve(plannedStepID)
		if plannedStepHealth.Status == api.HealthUnhealthy {
			h := flowStepHealth(plannedStepID, plannedStepHealth)
			r.cache[stepID] = h
			return h
		}
		if plannedStepHealth.Status == api.HealthUnknown &&
			plannedStepHealth.Error != "" && unknown == nil {
			unknown = flowStepHealth(plannedStepID, plannedStepHealth)
		}
	}

	if unknown != nil {
		r.cache[stepID] = unknown
		return unknown
	}

	healthy := &api.HealthState{Status: api.HealthHealthy}
	r.cache[stepID] = healthy
	return healthy
}

func baseHealth(
	stepID api.StepID, base map[api.StepID]*api.HealthState,
) *api.HealthState {
	if h, ok := base[stepID]; ok && h != nil {
		return h
	}
	return &api.HealthState{Status: api.HealthUnknown}
}

func flowStepHealth(
	stepID api.StepID, health *api.HealthState,
) *api.HealthState {
	if health == nil {
		return &api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step %s health unknown", stepID),
		}
	}
	switch health.Status {
	case api.HealthUnhealthy:
		if health.Error == "" {
			return &api.HealthState{
				Status: api.HealthUnhealthy,
				Error:  fmt.Sprintf("step %s unhealthy", stepID),
			}
		}
		return &api.HealthState{
			Status: api.HealthUnhealthy,
			Error:  fmt.Sprintf("step %s: %s", stepID, health.Error),
		}
	case api.HealthUnknown:
		if health.Error == "" {
			return &api.HealthState{
				Status: api.HealthUnknown,
				Error:  fmt.Sprintf("step %s health unknown", stepID),
			}
		}
		return &api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step %s: %s", stepID, health.Error),
		}
	default:
		return &api.HealthState{Status: api.HealthHealthy}
	}
}

func (r *healthResolver) previewFlowPlan(
	stepID api.StepID, step *api.Step,
) (*api.ExecutionPlan, error) {
	if plan, ok := r.plans[stepID]; ok {
		return plan, nil
	}
	if err, ok := r.planErrs[stepID]; ok {
		return nil, err
	}

	plan, err := plan.Create(r.catState, step.Flow.Goals, api.Args{})
	if err != nil {
		r.planErrs[stepID] = err
		return nil, err
	}

	r.plans[stepID] = plan
	return plan, nil
}
