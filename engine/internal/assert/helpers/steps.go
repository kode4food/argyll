package helpers

import (
	"uuid"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// NewTestStep creates a basic HTTP step for testing with required, optional, and output attributes
func NewTestStep() *api.Step {
	return &api.Step{
		ID:   api.StepID("test-step-" + uuid.New().String()[:8]),
		Name: "Test Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080/transform",
				Timeout:  30 * api.Second,
			},
		},
		Attributes: api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
			},
			"optional": {
				Role: api.RoleOptional,
			},
			"output": {
				Role: api.RoleOutput,
			},
		},
	}
}

// NewTestStepWithArgs creates an HTTP step with the specified required and optional input arguments
func NewTestStepWithArgs(required []api.Name, optional []api.Name) *api.Step {
	st := NewTestStep()

	st.Attributes = api.AttributeSpecs{}
	for _, arg := range required {
		st.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleRequired,
		}
	}

	for _, arg := range optional {
		st.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleOptional,
		}
	}

	return st
}

// NewSimpleStep creates a minimal HTTP step with the specified ID
func NewSimpleStep(id api.StepID) *api.Step {
	return &api.Step{
		ID:         id,
		Name:       "Test Step",
		Type:       api.StepTypeService,
		Attributes: api.AttributeSpecs{},
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
		},
	}
}

// NewStepWithOutputs creates an HTTP step that produces the specified output attributes
func NewStepWithOutputs(id api.StepID, outputs ...api.Name) *api.Step {
	st := NewSimpleStep(id)
	if st.Attributes == nil {
		st.Attributes = api.AttributeSpecs{}
	}
	for _, name := range outputs {
		st.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
		}
	}
	return st
}

// NewScriptStep creates a script step with the given language, code, and output attributes
func NewScriptStep(id api.StepID, language, src string, outputs ...api.Name) *api.Step {
	st := &api.Step{
		ID:   id,
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: language,
			Source:   src,
		},
		Attributes: api.AttributeSpecs{},
	}
	for _, name := range outputs {
		st.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
		}
	}
	return st
}

// NewStepWithPredicate creates an HTTP step with a predicate script that determines whether the
// step should execute
func NewStepWithPredicate(id api.StepID, lang, src string, outputs ...api.Name) *api.Step {
	st := NewSimpleStep(id)
	st.Predicate = &api.ScriptConfig{
		Language: lang,
		Source:   src,
	}
	if st.Attributes == nil {
		st.Attributes = api.AttributeSpecs{}
	}
	for _, name := range outputs {
		st.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
		}
	}
	return st
}
