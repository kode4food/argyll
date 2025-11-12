package builder

import (
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type Step struct {
	predicate  *api.ScriptConfig
	http       *api.HTTPConfig
	script     *api.ScriptConfig
	id         timebox.ID
	name       api.Name
	stepType   api.StepType
	version    string
	attributes map[api.Name]*api.AttributeSpec
	timeout    int64
}

var (
	camelCaseRegex = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	delimiterRegex = regexp.MustCompile(`[\s_]+`)
)

// NewStep creates a new step builder with the specified name
func NewStep(name api.Name) *Step {
	id := timebox.ID(toSnakeCase(string(name)))
	return &Step{
		id:         id,
		name:       name,
		version:    "1.0.0",
		stepType:   api.StepTypeSync,
		timeout:    30 * api.Second,
		attributes: map[api.Name]*api.AttributeSpec{},
	}
}

func (s *Step) WithID(id timebox.ID) *Step {
	res := *s
	res.id = id
	return &res
}

func (s *Step) Required(name api.Name, argType api.AttributeType) *Step {
	res := *s
	res.attributes = maps.Clone(s.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: argType,
	}
	return &res
}

func (s *Step) Optional(
	name api.Name, argType api.AttributeType, defaultValue string,
) *Step {
	res := *s
	res.attributes = maps.Clone(s.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role:    api.RoleOptional,
		Type:    argType,
		Default: defaultValue,
	}
	return &res
}

func (s *Step) Output(name api.Name, argType api.AttributeType) *Step {
	res := *s
	res.attributes = maps.Clone(s.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: argType,
	}
	return &res
}

// WithForEach marks an attribute as supporting multi work items (arrays)
func (s *Step) WithForEach(name api.Name) *Step {
	res := *s
	res.attributes = maps.Clone(s.attributes)
	if attr, ok := res.attributes[name]; ok {
		newAttr := *attr
		newAttr.ForEach = true
		res.attributes[name] = &newAttr
	}
	return &res
}

func (s *Step) WithPredicate(language, script string) *Step {
	res := *s
	res.predicate = &api.ScriptConfig{
		Language: language,
		Script:   script,
	}
	return &res
}

func (s *Step) WithAlePredicate(script string) *Step {
	return s.WithPredicate(api.ScriptLangAle, script)
}

func (s *Step) WithLuaPredicate(script string) *Step {
	return s.WithPredicate(api.ScriptLangLua, script)
}

func (s *Step) WithVersion(version string) *Step {
	res := *s
	res.version = version
	return &res
}

func (s *Step) WithEndpoint(endpoint string) *Step {
	res := *s
	if res.http == nil {
		res.http = &api.HTTPConfig{}
	} else {
		httpCopy := *res.http
		res.http = &httpCopy
	}
	res.http.Endpoint = endpoint
	if res.stepType == "" {
		res.stepType = api.StepTypeSync
	}
	return &res
}

func (s *Step) WithScript(script string) *Step {
	res := *s
	res.script = &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   script,
	}
	res.stepType = api.StepTypeScript
	return &res
}

func (s *Step) WithScriptLanguage(lang, script string) *Step {
	res := *s
	res.script = &api.ScriptConfig{
		Language: lang,
		Script:   script,
	}
	res.stepType = api.StepTypeScript
	return &res
}

func (s *Step) WithHealthCheck(endpoint string) *Step {
	res := *s
	if res.http == nil {
		res.http = &api.HTTPConfig{}
	} else {
		httpCopy := *res.http
		res.http = &httpCopy
	}
	res.http.HealthCheck = endpoint
	return &res
}

func (s *Step) WithTimeout(timeout int64) *Step {
	res := *s
	res.timeout = timeout
	return &res
}

func (s *Step) WithType(stepType api.StepType) *Step {
	res := *s
	res.stepType = stepType
	return &res
}

func (s *Step) WithAsyncExecution() *Step {
	res := *s
	res.stepType = api.StepTypeAsync
	return &res
}

func (s *Step) WithSyncExecution() *Step {
	res := *s
	res.stepType = api.StepTypeSync
	return &res
}

func (s *Step) WithScriptExecution() *Step {
	res := *s
	res.stepType = api.StepTypeScript
	return &res
}

func (s *Step) Build() (*api.Step, error) {
	var httpConfig *api.HTTPConfig
	if s.http != nil {
		httpCopy := *s.http
		httpCopy.Timeout = s.timeout
		httpConfig = &httpCopy
	}

	step := &api.Step{
		ID:         s.id,
		Name:       s.name,
		Type:       s.stepType,
		Attributes: s.attributes,
		Predicate:  s.predicate,
		Version:    s.version,
		HTTP:       httpConfig,
		Script:     s.script,
	}

	if err := step.Validate(); err != nil {
		return nil, err
	}

	return step, nil
}

func toSnakeCase(s string) string {
	s = camelCaseRegex.ReplaceAllString(s, "$1-$2")
	s = delimiterRegex.ReplaceAllString(s, "-")
	return strings.ToLower(s)
}
