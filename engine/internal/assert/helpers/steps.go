package helpers

import (
	"github.com/google/uuid"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// NewTestStep creates a basic HTTP step for testing with required, optional,
// and output attributes
func NewTestStep() *api.Step {
	return &api.Step{
		ID:   api.StepID("test-step-" + uuid.New().String()[:8]),
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080/transform",
			Timeout:  30 * api.Second,
		},
		Attributes: api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role: api.RoleOptional,
				Type: api.TypeString,
			},
			"output": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		},
	}
}

// NewTestStepWithArgs creates an HTTP step with the specified required and
// optional input arguments
func NewTestStepWithArgs(required []api.Name, optional []api.Name) *api.Step {
	st := NewTestStep()

	st.Attributes = api.AttributeSpecs{}
	for _, arg := range required {
		st.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
	}

	for _, arg := range optional {
		st.Attributes[arg] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
		}
	}

	return st
}

// NewSimpleStep creates a minimal HTTP step with the specified ID
func NewSimpleStep(id api.StepID) *api.Step {
	return &api.Step{
		ID:         id,
		Name:       "Test Step",
		Type:       api.StepTypeSync,
		Attributes: api.AttributeSpecs{},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}
}

// NewStepWithOutputs creates an HTTP step that produces the specified output
// attributes
func NewStepWithOutputs(id api.StepID, outputs ...api.Name) *api.Step {
	st := NewSimpleStep(id)
	if st.Attributes == nil {
		st.Attributes = api.AttributeSpecs{}
	}
	for _, name := range outputs {
		st.Attributes[name] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}
	}
	return st
}

// NewScriptStep creates a script-based step with the specified language, code,
// and output attributes
func NewScriptStep(
	id api.StepID, language, script string, outputs ...api.Name,
) *api.Step {
	st := &api.Step{
		ID:   id,
		Name: "Script Step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: language,
			Script:   script,
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

// NewStepWithPredicate creates an HTTP step with a predicate script that
// determines whether the step should execute
func NewStepWithPredicate(
	id api.StepID, lang, script string, outputs ...api.Name,
) *api.Step {
	st := NewSimpleStep(id)
	st.Predicate = &api.ScriptConfig{
		Language: lang,
		Script:   script,
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
