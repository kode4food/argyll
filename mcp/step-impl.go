package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/mcp/internal/guidance"
)

type stepImplResult struct {
	StepName            api.Name       `json:"step_name"`
	StepType            api.StepType   `json:"step_type"`
	ActionMode          api.ActionMode `json:"action_mode,omitempty"`
	Method              string         `json:"method"`
	Code                string         `json:"code"`
	Inputs              []api.Name     `json:"inputs"`
	Outputs             []api.Name     `json:"outputs"`
	ImplementationNotes []string       `json:"implementation_notes"`
	Registration        *api.Step      `json:"registration"`
}

const registrationLiteralWidth = 90

func (s *Server) generateStepImpl(
	_ context.Context, args generateStepImplArgs,
) (any, error) {
	if len(args.Step) == 0 {
		return nil, errInvalidParams("step is required")
	}

	lang := strings.ToLower(strings.TrimSpace(args.Language))
	if lang != "go" && lang != "python" {
		return nil, errInvalidParams("language must be go or python")
	}
	raw, err := json.Marshal(args.Step)
	if err != nil {
		return nil, err
	}
	var st api.Step
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	if st.ID == "" || st.Name == "" {
		return nil, errInvalidParams("step id and name are required")
	}
	switch st.Type {
	case api.StepTypeService, api.StepTypeScript, api.StepTypeFlow, api.StepTypeAgent:
	default:
		return nil, errInvalidParams("step type must be service, script, flow, or agent")
	}
	if st.Type == api.StepTypeScript {
		if st.Script == nil || (st.Script.Language != api.ScriptLangLua &&
			st.Script.Language != api.ScriptLangArgyll) {
			return nil, errInvalidParams("script_language must be lua or argyll for script steps")
		}
	}
	inputs, outputs := stepIO(&st)
	var actionMode api.ActionMode
	var method string
	if st.HTTP != nil {
		actionMode = st.HTTP.Invoke.Mode
		method = st.HTTP.Invoke.Method
	}
	code, err := guidance.RenderTemplate(lang+"-registration.tmpl", registrationChunks(string(raw)))
	if err != nil {
		return nil, err
	}

	return stepImplResult{
		StepName:            st.Name,
		StepType:            st.Type,
		ActionMode:          actionMode,
		Method:              method,
		Inputs:              inputs,
		Outputs:             outputs,
		ImplementationNotes: stepImplNotes(st.HTTP != nil, method, inputs, outputs),
		Code:                code,
		Registration:        &st,
	}, nil
}

func stepIO(st *api.Step) ([]api.Name, []api.Name) {
	var inputs []api.Name
	var outputs []api.Name
	for name, attr := range st.Attributes {
		switch attr.Role {
		case api.RoleRequired, api.RoleOptional:
			inputs = append(inputs, name)
		case api.RoleOutput:
			outputs = append(outputs, name)
		}
	}
	slices.Sort(inputs)
	slices.Sort(outputs)
	return inputs, outputs
}

func registrationChunks(raw string) []string {
	var chunks []string
	var chunk string
	for _, r := range raw {
		next := chunk + string(r)
		if len(strconv.Quote(next)) > registrationLiteralWidth {
			chunks = append(chunks, chunk)
			chunk = string(r)
			continue
		}
		chunk = next
	}
	return append(chunks, chunk)
}

func stepImplNotes(isExternal bool, method string, inputs, outputs []api.Name) []string {
	res := []string{
		"This is registration code preserving the supplied Step, not a service implementation. " +
			"Implement and deploy the configured endpoints before registering service Steps. " +
			"Use sdk_step_template for a separate handler scaffold",
		"The registration is unchanged, including omitted fields; the engine validates it. " +
			"Go SDK registration can update an existing Step; Python sends a create request",
	}
	if isExternal {
		res = append(res,
			"Use the service contract already embedded in the proposed step instead of inventing "+
				"a new transport.",
		)
	}
	if method != "" && method != "POST" {
		res = append(res,
			"This step uses a non-POST HTTP method; map arguments to path or query fields "+
				"carefully.",
		)
	}
	if len(inputs) == 0 {
		res = append(res, "This draft has no inputs; verify that.")
	}
	if len(outputs) == 0 {
		res = append(res, "This draft has no outputs; verify that.")
	}
	return res
}
