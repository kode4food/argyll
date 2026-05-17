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
		Types   map[string]string
		Service string
		ID      string
		Name    string
		Source  string
		Inputs  []string
		Outputs []string
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
		Kind          string   `json:"kind"`
		Attributes    []string `json:"attributes"`
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
	nodes := make([]serviceNode, 0, len(spec.Operations))
	for _, op := range spec.Operations {
		inputs, outputs, types := operationIO(op)
		nodes = append(nodes, serviceNode{
			Service: name,
			ID:      op.ID,
			Name:    op.Summary,
			Source:  "contract",
			Inputs:  inputs,
			Outputs: outputs,
			Types:   types,
		})
	}
	return nodes
}

func operationIO(op openapi.Operation) ([]string, []string, map[string]string) {
	res := map[string]string{}
	var inputs []string
	var outputs []string
	for _, arg := range op.Inputs {
		if !arg.Required {
			continue
		}
		inputs = append(inputs, arg.Name)
		res[arg.Name] = coalesceType(arg.Type)
	}
	for _, arg := range op.Outputs {
		outputs = append(outputs, arg.Name)
		res[arg.Name] = coalesceType(arg.Type)
	}
	slices.Sort(inputs)
	slices.Sort(outputs)
	return inputs, outputs, res
}

func relationships(nodes []serviceNode) []relationship {
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
