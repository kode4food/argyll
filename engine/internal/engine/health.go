package engine

import (
	"fmt"
	"maps"
	"sort"

	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type healthResolver struct {
	cat      api.CatalogState
	steps    api.Steps
	base     map[api.StepID]api.HealthState
	cache    map[api.StepID]api.HealthState
	visiting map[api.StepID]bool
	plans    map[api.StepID]*api.ExecutionPlan
	planErrs map[api.StepID]error
}

// UpdateStepHealth updates the health status of a registered step, used
// primarily for tracking HTTP service availability and script errors
func (e *Engine) UpdateStepHealth(
	stepID api.StepID, health api.HealthStatus, errMsg string,
) error {
	nid := e.LocalNodeID()
	cmd := func(st api.ClusterState, ag *ClusterAggregator) error {
		node := st.Nodes[nid]
		if h, ok := node.Health[stepID]; ok {
			if h.Status == health && h.Error == errMsg {
				return nil
			}
		}

		return events.Raise(ag, api.EventTypeStepHealthChanged,
			api.StepHealthChangedEvent{
				NodeID: nid,
				StepID: stepID,
				Status: health,
				Error:  errMsg,
			},
		)
	}

	_, err := e.execCluster(cmd)
	if err != nil {
		return err
	}

	e.setLocalHealth(stepID, api.HealthState{
		Status: health,
		Error:  errMsg,
	})
	return nil
}

// ResolveHealth returns resolved health for all steps, deriving flow step
// health from all steps included in the flow's execution preview
func ResolveHealth(
	cat api.CatalogState, base map[api.StepID]api.HealthState,
) map[api.StepID]api.HealthState {
	resolver := &healthResolver{
		cat:      cat,
		steps:    cat.Steps,
		base:     base,
		cache:    map[api.StepID]api.HealthState{},
		visiting: map[api.StepID]bool{},
		plans:    map[api.StepID]*api.ExecutionPlan{},
		planErrs: map[api.StepID]error{},
	}

	resolved := make(map[api.StepID]api.HealthState, len(cat.Steps))
	for sid := range cat.Steps {
		resolved[sid] = resolver.resolve(sid)
	}
	return resolved
}

// MergeNodeHealth reduces per-node step health into a cluster-wide worst-case
// view while preserving stable results
func MergeNodeHealth(cluster api.ClusterState) map[api.StepID]api.HealthState {
	res := map[api.StepID]api.HealthState{}
	nodes := make([]string, 0, len(cluster.Nodes))
	for id := range cluster.Nodes {
		nodes = append(nodes, string(id))
	}
	sort.Strings(nodes)

	for _, rawNodeID := range nodes {
		nid := api.NodeID(rawNodeID)
		node := cluster.Nodes[nid]
		steps := make([]string, 0, len(node.Health))
		for sid := range node.Health {
			steps = append(steps, string(sid))
		}
		sort.Strings(steps)

		for _, rawStepID := range steps {
			sid := api.StepID(rawStepID)
			res[sid] = mergeHealthState(
				nid, res[sid], node.Health[sid],
			)
		}
	}

	return res
}

func (e *Engine) canDispatchLocally(stepID api.StepID) bool {
	h, ok := e.getLocalHealth(stepID)
	if !ok {
		return true
	}

	return h.Status != api.HealthUnhealthy
}

func (e *Engine) loadLocalHealth() error {
	st, err := e.clusterExec.Get(events.ClusterKey)
	if err != nil {
		return err
	}
	st = e.withConfiguredNodes(st)

	node := st.Nodes[e.LocalNodeID()]
	health := map[api.StepID]api.HealthState{}
	maps.Copy(health, node.Health)

	e.healthMu.Lock()
	e.health = health
	e.healthMu.Unlock()
	return nil
}

func (e *Engine) getLocalHealth(stepID api.StepID) (api.HealthState, bool) {
	e.healthMu.RLock()
	defer e.healthMu.RUnlock()

	h, ok := e.health[stepID]
	return h, ok
}

func (e *Engine) setLocalHealth(stepID api.StepID, h api.HealthState) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	e.health[stepID] = h
}

func (r *healthResolver) resolve(stepID api.StepID) api.HealthState {
	if h, ok := r.cache[stepID]; ok {
		return h
	}

	if base, ok := r.base[stepID]; ok {
		if base.Status != api.HealthUnknown {
			r.cache[stepID] = base
			return base
		}
	}

	step, ok := r.steps[stepID]
	if !ok {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step not found: %s", stepID),
		}
		r.cache[stepID] = h
		return h
	}

	if r.visiting[stepID] {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow health cycle at step %s", stepID),
		}
		r.cache[stepID] = h
		return h
	}

	if step.Type != api.StepTypeFlow || step.Flow == nil {
		h := defaultStepHealth(step, stepID, r.base)
		r.cache[stepID] = h
		return h
	}

	r.visiting[stepID] = true
	defer delete(r.visiting, stepID)

	pl, err := r.previewFlowPlan(stepID, step)
	if err != nil {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow preview failed for %s: %v", stepID, err),
		}
		r.cache[stepID] = h
		return h
	}

	var unknown api.HealthState
	for id := range pl.Steps {
		health := r.resolve(id)
		if health.Status == api.HealthUnhealthy {
			h := flowStepHealth(id, health)
			r.cache[stepID] = h
			return h
		}
		if health.Status == api.HealthUnknown &&
			health.Error != "" && unknown == (api.HealthState{}) {
			unknown = flowStepHealth(id, health)
		}
	}

	if unknown != (api.HealthState{}) {
		r.cache[stepID] = unknown
		return unknown
	}

	healthy := api.HealthState{Status: api.HealthHealthy}
	r.cache[stepID] = healthy
	return healthy
}

func baseHealth(
	stepID api.StepID, base map[api.StepID]api.HealthState,
) api.HealthState {
	if h, ok := base[stepID]; ok {
		return h
	}
	return api.HealthState{Status: api.HealthUnknown}
}

func defaultStepHealth(
	step *api.Step, stepID api.StepID, base map[api.StepID]api.HealthState,
) api.HealthState {
	if step.Type == api.StepTypeScript {
		return scriptHealth(stepID, base)
	}
	return baseHealth(stepID, base)
}

func scriptHealth(
	stepID api.StepID, base map[api.StepID]api.HealthState,
) api.HealthState {
	if h, ok := base[stepID]; ok {
		if h.Status == api.HealthUnknown && h.Error == "" {
			return api.HealthState{Status: api.HealthHealthy}
		}
		return h
	}
	return api.HealthState{Status: api.HealthHealthy}
}

func flowStepHealth(
	stepID api.StepID, health api.HealthState,
) api.HealthState {
	switch health.Status {
	case api.HealthUnhealthy:
		if health.Error == "" {
			return api.HealthState{
				Status: api.HealthUnhealthy,
				Error:  fmt.Sprintf("step %s unhealthy", stepID),
			}
		}
		return api.HealthState{
			Status: api.HealthUnhealthy,
			Error:  fmt.Sprintf("step %s: %s", stepID, health.Error),
		}
	case api.HealthUnknown:
		if health.Error == "" {
			return api.HealthState{
				Status: api.HealthUnknown,
				Error:  fmt.Sprintf("step %s health unknown", stepID),
			}
		}
		return api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step %s: %s", stepID, health.Error),
		}
	default:
		return api.HealthState{Status: api.HealthHealthy}
	}
}

func mergeHealthState(
	nodeID api.NodeID, curr, next api.HealthState,
) api.HealthState {
	norm := api.HealthState{
		Status: next.Status,
		Error:  annotateHealthError(string(nodeID), next.Error),
	}
	if curr == (api.HealthState{}) {
		return norm
	}

	if healthRank(norm.Status) > healthRank(curr.Status) {
		return norm
	}
	if healthRank(norm.Status) < healthRank(curr.Status) {
		return curr
	}
	if curr.Error == "" && norm.Error != "" {
		return norm
	}
	return curr
}

func annotateHealthError(nodeID, errMsg string) string {
	if errMsg == "" {
		return ""
	}
	return fmt.Sprintf("node %s: %s", nodeID, errMsg)
}

func healthRank(st api.HealthStatus) int {
	switch st {
	case api.HealthUnhealthy:
		return 2
	case api.HealthUnknown:
		return 1
	default:
		return 0
	}
}

func (r *healthResolver) previewFlowPlan(
	stepID api.StepID, step *api.Step,
) (*api.ExecutionPlan, error) {
	if pl, ok := r.plans[stepID]; ok {
		return pl, nil
	}
	if err, ok := r.planErrs[stepID]; ok {
		return nil, err
	}

	pl, err := plan.Create(r.cat, step.Flow.Goals, api.Args{})
	if err != nil {
		r.planErrs[stepID] = err
		return nil, err
	}

	r.plans[stepID] = pl
	return pl, nil
}
