package mcp

import (
	"cmp"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	serviceNode struct {
		Types   map[api.Name]string
		Service string
		ID      api.StepID
		Name    api.Name
		Source  string
		Inputs  []api.Name
		Outputs []api.Name
	}

	catalogStep struct {
		ID       api.StepID `json:"id"`
		Name     api.Name   `json:"name"`
		Source   string     `json:"source"`
		Path     string     `json:"path,omitempty"`
		Required []api.Name `json:"required,omitempty"`
		Optional []api.Name `json:"optional,omitempty"`
		Outputs  []api.Name `json:"outputs,omitempty"`
	}

	relationship struct {
		SourceService string     `json:"source_service"`
		SourceStepID  api.StepID `json:"source_step_id"`
		TargetService string     `json:"target_service"`
		TargetStepID  api.StepID `json:"target_step_id"`
		Kind          string     `json:"kind"`
		Attributes    []api.Name `json:"attributes"`
	}

	missingAttr struct {
		Service    string     `json:"service"`
		StepID     api.StepID `json:"step_id"`
		StepName   api.Name   `json:"step_name"`
		Attribute  api.Name   `json:"attribute"`
		Type       string     `json:"type,omitempty"`
		Kind       string     `json:"kind"`
		Confidence string     `json:"confidence"`
	}

	missingAttrKey struct {
		Service   string
		StepID    api.StepID
		Attribute api.Name
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
		nodes = append(nodes, serviceNode{
			Service: "registered",
			ID:      st.ID,
			Name:    st.Name,
			Source:  st.Source,
			Inputs:  slices.Clone(st.Required),
			Outputs: slices.Clone(st.Outputs),
		})
	}
	return nodes
}

func serviceNodes(name string, spec openapi.Result) []serviceNode {
	nodes := make([]serviceNode, 0, len(spec.Operations))
	for _, op := range spec.Operations {
		io := operationIO(op)
		nodes = append(nodes, serviceNode{
			Service: name,
			ID:      op.ID,
			Name:    api.Name(op.Summary),
			Source:  "contract",
			Inputs:  io.inputs,
			Outputs: io.outputs,
			Types:   io.types,
		})
	}
	return nodes
}

type operationIORes struct {
	inputs  []api.Name
	outputs []api.Name
	types   map[api.Name]string
}

func operationIO(op openapi.Operation) operationIORes {
	res := map[api.Name]string{}
	var inputs []api.Name
	var outputs []api.Name
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
	return operationIORes{
		inputs:  inputs,
		outputs: outputs,
		types:   res,
	}
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
		return cmp.Compare(a.SourceStepID, b.SourceStepID)
	})
	return res
}

func missingAttributes(nodes []serviceNode) []missingAttr {
	provided := map[api.Name]string{}
	for _, node := range nodes {
		for _, out := range node.Outputs {
			provided[out] = node.Types[out]
		}
	}

	seen := util.Set[missingAttrKey]{}
	var res []missingAttr
	for _, node := range nodes {
		for _, in := range node.Inputs {
			if _, ok := provided[in]; ok {
				continue
			}
			key := missingAttrKey{
				Service:   node.Service,
				StepID:    node.ID,
				Attribute: in,
			}
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
