package mcp

import (
	"maps"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	serviceNode struct {
		Service string
		ID      string
		Name    string
		Source  string
		Inputs  []string
		Outputs []string
		Types   map[string]string
		Step    map[string]any
	}

	catalogStep struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Source   string   `json:"source"`
		Path     string   `json:"path,omitempty"`
		Required []string `json:"required,omitempty"`
		Optional []string `json:"optional,omitempty"`
		Outputs  []string `json:"outputs,omitempty"`
	}

	relationship struct {
		SourceService string   `json:"source_service"`
		SourceStepID  string   `json:"source_step_id"`
		TargetService string   `json:"target_service"`
		TargetStepID  string   `json:"target_step_id"`
		Attributes    []string `json:"attributes"`
		Kind          string   `json:"kind"`
	}

	missingAttr struct {
		Service    string `json:"service"`
		StepID     string `json:"step_id"`
		StepName   string `json:"step_name"`
		Attribute  string `json:"attribute"`
		Type       string `json:"type,omitempty"`
		Kind       string `json:"kind"`
		Confidence string `json:"confidence"`
	}

	bridgeOpportunity struct {
		Kind           string   `json:"kind"`
		SourceService  string   `json:"source_service"`
		SourceStepID   string   `json:"source_step_id"`
		SourceStepName string   `json:"source_step_name,omitempty"`
		SourceAttr     string   `json:"source_attribute"`
		SourceType     string   `json:"source_type,omitempty"`
		TargetService  string   `json:"target_service"`
		TargetStepID   string   `json:"target_step_id"`
		TargetStepName string   `json:"target_step_name,omitempty"`
		TargetAttr     string   `json:"target_attribute"`
		TargetType     string   `json:"target_type,omitempty"`
		SharedKeys     []string `json:"shared_keys,omitempty"`
		Confidence     string   `json:"confidence"`
		Rationale      string   `json:"rationale"`
	}
)

func existingCatalogPayload(steps []openapi.Step) []catalogStep {
	res := make([]catalogStep, 0, len(steps))
	for _, st := range steps {
		res = append(res, catalogStep{
			ID:       st.ID,
			Name:     st.Name,
			Source:   st.Source,
			Path:     st.Path,
			Required: st.Required,
			Optional: st.Optional,
			Outputs:  st.Outputs,
		})
	}
	return res
}

func landscapeNodes(existing []openapi.Step) []serviceNode {
	nodes := make([]serviceNode, 0, len(existing))
	for _, st := range existing {
		types := map[string]string{}
		maps.Copy(types, st.InputsByType)
		maps.Copy(types, st.OutputsByType)
		nodes = append(nodes, serviceNode{
			Service: "registered",
			ID:      st.ID,
			Name:    st.Name,
			Source:  st.Source,
			Inputs:  slices.Clone(st.Required),
			Outputs: slices.Clone(st.Outputs),
			Types:   types,
		})
	}
	return nodes
}

func serviceNodes(name string, spec openapi.Result) []serviceNode {
	nodes := make([]serviceNode, 0, len(spec.RecommendedSteps))
	for _, st := range spec.RecommendedSteps {
		nodes = append(nodes, serviceNode{
			Service: name,
			ID:      st.ID,
			Name:    st.Name,
			Source:  st.Source,
			Inputs:  slices.Clone(st.Required),
			Outputs: slices.Clone(st.Outputs),
			Types:   stepTypes(st.Step),
			Step:    cloneMap(st.Step),
		})
	}
	return nodes
}

func inferRelationships(nodes []serviceNode) []relationship {
	var res []relationship
	for _, src := range nodes {
		for _, dst := range nodes {
			if src.ID == dst.ID || src.Service == dst.Service {
				continue
			}
			attrs := sharedStrings(src.Outputs, dst.Inputs)
			if len(attrs) == 0 {
				continue
			}
			res = append(res, relationship{
				SourceService: src.Service,
				SourceStepID:  src.ID,
				TargetService: dst.Service,
				TargetStepID:  dst.ID,
				Attributes:    attrs,
				Kind:          "direct_dependency",
			})
		}
	}
	slices.SortFunc(res, func(a, b relationship) int {
		return strings.Compare(a.SourceStepID, b.SourceStepID)
	})
	return res
}

func missingAttributes(nodes []serviceNode) []missingAttr {
	provided := map[string]string{}
	for _, node := range nodes {
		for _, out := range node.Outputs {
			provided[out] = node.Types[out]
		}
	}

	seen := util.Set[string]{}
	var res []missingAttr
	for _, node := range nodes {
		for _, in := range node.Inputs {
			if _, ok := provided[in]; ok {
				continue
			}
			key := node.Service + "|" + node.ID + "|" + in
			if seen.Contains(key) {
				continue
			}
			seen.Add(key)
			res = append(res, missingAttr{
				Service:    node.Service,
				StepID:     node.ID,
				StepName:   node.Name,
				Attribute:  in,
				Type:       node.Types[in],
				Kind:       "unprovided_required_input",
				Confidence: "high",
			})
		}
	}
	return res
}

func bridgeOpportunities(
	nodes []serviceNode, missing []missingAttr,
) []bridgeOpportunity {
	var res []bridgeOpportunity
	for _, miss := range missing {
		for _, node := range nodes {
			if node.Service == miss.Service {
				continue
			}
			res = append(res, bridgesFromNode(node, miss)...)
		}
	}
	return dedupeBridgeOpportunities(res)
}

func bridgesFromNode(node serviceNode, miss missingAttr) []bridgeOpportunity {
	var res []bridgeOpportunity
	for _, out := range node.Outputs {
		if !scriptBridgeMatch(node, out, miss) {
			continue
		}
		res = append(res, bridgeOpportunity{
			Kind:           "script_bridge",
			SourceService:  node.Service,
			SourceStepID:   node.ID,
			SourceStepName: node.Name,
			SourceAttr:     out,
			SourceType:     node.Types[out],
			TargetService:  miss.Service,
			TargetStepID:   miss.StepID,
			TargetStepName: miss.StepName,
			TargetAttr:     miss.Attribute,
			TargetType:     miss.Type,
			SharedKeys:     sharedKeys(node.Inputs, []string{miss.Attribute}),
			Confidence:     scriptBridgeConfidence(node.Types[out], miss.Type),
			Rationale:      scriptBridgeRationale(node, out, miss),
		})
	}
	return res
}

func dedupeBridgeOpportunities(items []bridgeOpportunity) []bridgeOpportunity {
	seen := util.Set[string]{}
	var res []bridgeOpportunity
	for _, item := range items {
		key := item.Kind + "|" + item.SourceStepID + "|" + item.SourceAttr +
			"|" + item.TargetStepID + "|" + item.TargetAttr
		if seen.Contains(key) {
			continue
		}
		seen.Add(key)
		res = append(res, item)
	}
	return res
}

func scriptBridgeMatch(
	node serviceNode, source string, target missingAttr,
) bool {
	if sameType(node.Types[source], target.Type) {
		return false
	}
	if source == target.Attribute {
		return true
	}
	return tokenOverlap(source, target.Attribute) > 0
}

func scriptBridgeConfidence(sourceType, targetType string) string {
	if sourceType == "" || targetType == "" {
		return "medium"
	}
	return "high"
}

func scriptBridgeRationale(
	node serviceNode, source string, target missingAttr,
) string {
	if source == target.Attribute {
		return "source output matches the missing input name but needs a " +
			"Lua transform because the types differ"
	}
	if len(sharedKeys(node.Inputs, []string{target.Attribute})) != 0 {
		return "source output and missing input share a planning key but " +
			"need a Lua transform to bridge the type gap"
	}
	return "source output and missing input share planner tokens but " +
		"need a Lua transform to bridge the shape gap"
}
