package engine

import (
	"fmt"
	"maps"
	"sort"

	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type healthResolver struct {
	eng      *Engine
	cat      api.CatalogState
	steps    api.Steps
	base     map[api.StepID]api.HealthState
	cache    map[api.StepID]api.HealthState
	visiting map[api.StepID]bool
	plans    map[api.StepID]*api.ExecutionPlan
	planErrs map[api.StepID]error
	match    policy.Matcher
}

// UpdateStepHealth updates the health status of a registered step, used
// primarily for tracking HTTP service availability and script errors
func (e *Engine) UpdateStepHealth(
	sid api.StepID, health api.HealthStatus, errMsg string,
) error {
	nid := e.LocalNodeID()
	cmd := func(st api.ClusterState, ag *ClusterAggregator) error {
		node := st.Nodes[nid]
		if h, ok := node.Health[sid]; ok {
			if h.Status == health && h.Error == errMsg {
				return nil
			}
		}

		return events.Raise(ag, api.EventTypeStepHealthChanged,
			api.StepHealthChangedEvent{
				NodeID: nid,
				StepID: sid,
				Status: health,
				Error:  errMsg,
			},
		)
	}

	_, err := e.execCluster(cmd)
	if err != nil {
		return err
	}

	e.setLocalHealth(sid, api.HealthState{
		Status: health,
		Error:  errMsg,
	})
	return nil
}

// ResolveHealth returns resolved health for all steps, deriving flow step
// health from all steps included in the flow's execution preview
func (e *Engine) ResolveHealth(
	match policy.Matcher, cat api.CatalogState,
	base map[api.StepID]api.HealthState,
) map[api.StepID]api.HealthState {
	resolver := &healthResolver{
		eng:      e,
		cat:      cat,
		steps:    cat.Steps,
		base:     base,
		cache:    map[api.StepID]api.HealthState{},
		visiting: map[api.StepID]bool{},
		plans:    map[api.StepID]*api.ExecutionPlan{},
		planErrs: map[api.StepID]error{},
		match:    match,
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
			res[sid] = mergeHealthState(nid, res[sid], node.Health[sid])
		}
	}

	return res
}

func (e *Engine) canDispatchLocally(sid api.StepID) bool {
	h, ok := e.getLocalHealth(sid)
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
	h := map[api.StepID]api.HealthState{}
	maps.Copy(h, node.Health)

	e.healthMu.Lock()
	e.health = h
	e.healthMu.Unlock()
	return nil
}

func (e *Engine) getLocalHealth(sid api.StepID) (api.HealthState, bool) {
	e.healthMu.RLock()
	defer e.healthMu.RUnlock()

	h, ok := e.health[sid]
	return h, ok
}

func (e *Engine) setLocalHealth(sid api.StepID, h api.HealthState) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	e.health[sid] = h
}

func (r *healthResolver) resolve(sid api.StepID) api.HealthState {
	if h, ok := r.cache[sid]; ok {
		return h
	}

	if base, ok := r.base[sid]; ok {
		if base.Status != api.HealthUnknown {
			r.cache[sid] = base
			return base
		}
	}

	step, ok := r.steps[sid]
	if !ok {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step not found: %s", sid),
		}
		r.cache[sid] = h
		return h
	}

	if r.visiting[sid] {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow health cycle at step %s", sid),
		}
		r.cache[sid] = h
		return h
	}

	children, err := r.eng.steps.Children(step)
	if err != nil {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  err.Error(),
		}
		r.cache[sid] = h
		return h
	}
	if len(children) == 0 {
		h := r.resolveStepHealth(step, sid)
		r.cache[sid] = h
		return h
	}

	r.visiting[sid] = true
	defer delete(r.visiting, sid)

	pl, err := r.previewFlowPlan(sid, children)
	if err != nil {
		h := api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("flow preview failed for %s: %v", sid, err),
		}
		r.cache[sid] = h
		return h
	}

	var unknown api.HealthState
	for id := range pl.Steps {
		h := r.resolve(id)
		if h.Status == api.HealthUnhealthy {
			h := flowStepHealth(id, h)
			r.cache[sid] = h
			return h
		}
		if h.Status == api.HealthUnknown && h.Error != "" &&
			unknown == (api.HealthState{}) {
			unknown = flowStepHealth(id, h)
		}
	}

	if unknown != (api.HealthState{}) {
		r.cache[sid] = unknown
		return unknown
	}

	healthy := api.HealthState{Status: api.HealthHealthy}
	r.cache[sid] = healthy
	return healthy
}

func (r *healthResolver) resolveStepHealth(
	st *api.Step, sid api.StepID,
) api.HealthState {
	if h, ok := r.base[sid]; ok {
		if h.Status != api.HealthUnknown || h.Error != "" {
			return h
		}
	}
	h, err := r.eng.steps.Health(st)
	if err != nil {
		return api.HealthState{
			Status: api.HealthUnknown,
			Error:  err.Error(),
		}
	}
	return h
}

func (r *healthResolver) previewFlowPlan(
	sid api.StepID, children []api.StepID,
) (*api.ExecutionPlan, error) {
	if pl, ok := r.plans[sid]; ok {
		return pl, nil
	}
	if err, ok := r.planErrs[sid]; ok {
		return nil, err
	}

	steps := r.cat.Steps
	st := r.steps[sid]
	if st.Flow != nil && st.Flow.SpaceID != "" {
		if _, ok := r.cat.Spaces[st.Flow.SpaceID]; !ok {
			return nil, fmt.Errorf("%w: %s", plan.ErrSpaceNotFound,
				st.Flow.SpaceID)
		}
		steps = r.cat.SpaceSteps(st.Flow.SpaceID)
	}
	pl, err := plan.Create(&plan.Request{
		Match:   r.match,
		Catalog: r.cat,
		Steps:   steps,
		Goals:   children,
		Init:    api.InitArgs{},
	})
	if err != nil {
		r.planErrs[sid] = err
		return nil, err
	}

	r.plans[sid] = pl
	return pl, nil
}

func flowStepHealth(sid api.StepID, h api.HealthState) api.HealthState {
	switch h.Status {
	case api.HealthUnhealthy:
		if h.Error == "" {
			return api.HealthState{
				Status: api.HealthUnhealthy,
				Error:  fmt.Sprintf("step %s unhealthy", sid),
			}
		}
		return api.HealthState{
			Status: api.HealthUnhealthy,
			Error:  fmt.Sprintf("step %s: %s", sid, h.Error),
		}
	case api.HealthUnknown:
		if h.Error == "" {
			return api.HealthState{
				Status: api.HealthUnknown,
				Error:  fmt.Sprintf("step %s health unknown", sid),
			}
		}
		return api.HealthState{
			Status: api.HealthUnknown,
			Error:  fmt.Sprintf("step %s: %s", sid, h.Error),
		}
	default:
		return api.HealthState{Status: api.HealthHealthy}
	}
}

func mergeHealthState(
	nid api.NodeID, curr, next api.HealthState,
) api.HealthState {
	norm := api.HealthState{
		Status: next.Status,
		Error:  annotateHealthError(string(nid), next.Error),
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
