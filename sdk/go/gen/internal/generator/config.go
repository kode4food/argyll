package generator

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type setter[T any] func(*T, string) error

const (
	nameProp    = "name"
	timeoutProp = "timeout"

	backoffTypeProp = "backoff_type"
	maxRetriesProp  = "max_retries"
	initBackoffProp = "init_backoff"
	maxBackoffProp  = "max_backoff"
	parallelismProp = "parallelism"

	roleProp        = "role"
	defaultProp     = "default"
	valueProp       = "value"
	keyProp         = "key"
	collectProp     = "collect"
	deadlineProp    = "deadline"
	forEachProp     = "for_each"
	mappingProp     = "mapping"
	compensatedProp = "compensated"
)

var (
	ErrBadProp = errors.New("invalid argyll property")
)

var stepSetters = map[string]setter[api.Step]{
	nameProp: func(s *api.Step, v string) error {
		s.Name = api.Name(v)
		return nil
	},
}

var httpSetters = map[string]setter[api.HTTPConfig]{
	timeoutProp: func(h *api.HTTPConfig, v string) error {
		ms, err := parseMillis(Option{Key: timeoutProp, Value: v})
		h.Invoke.Timeout = ms
		return err
	},
}

var workSetters = map[string]setter[api.WorkConfig]{
	backoffTypeProp: func(w *api.WorkConfig, v string) error {
		w.BackoffType = v
		return nil
	},
	maxRetriesProp: func(w *api.WorkConfig, v string) error {
		n, err := parseInt(Option{Key: maxRetriesProp, Value: v})
		if err != nil {
			return err
		}
		w.MaxRetries = n
		return nil
	},
	initBackoffProp: func(w *api.WorkConfig, v string) error {
		ms, err := parseMillis(Option{Key: initBackoffProp, Value: v})
		if err != nil {
			return err
		}
		w.InitBackoff = ms
		return nil
	},
	maxBackoffProp: func(w *api.WorkConfig, v string) error {
		ms, err := parseMillis(Option{Key: maxBackoffProp, Value: v})
		if err != nil {
			return err
		}
		w.MaxBackoff = ms
		return nil
	},
	parallelismProp: func(w *api.WorkConfig, v string) error {
		n, err := parseInt(Option{Key: parallelismProp, Value: v})
		if err != nil {
			return err
		}
		w.Parallelism = n
		return nil
	},
}

// applied after the role, so each setter finds the config it writes into
var attrSetters = map[string]setter[api.AttributeSpec]{
	compensatedProp: func(s *api.AttributeSpec, v string) error {
		on, err := parseFlag(Option{Key: compensatedProp, Value: v})
		s.Compensated = on
		return err
	},
	defaultProp: func(s *api.AttributeSpec, v string) error {
		if s.Optional == nil {
			return roleError(defaultProp, api.RoleOptional)
		}
		s.Optional.Default = jsonValue(s.Type, v)
		return nil
	},
	valueProp: func(s *api.AttributeSpec, v string) error {
		if s.Const == nil {
			return roleError(valueProp, api.RoleConst)
		}
		s.Const.Value = jsonValue(s.Type, v)
		return nil
	},
	keyProp: func(s *api.AttributeSpec, v string) error {
		if s.Meta == nil {
			return roleError(keyProp, api.RoleMeta)
		}
		s.Meta.Key = v
		return nil
	},
	deadlineProp: func(s *api.AttributeSpec, v string) error {
		if s.Optional == nil {
			return roleError(deadlineProp, api.RoleOptional)
		}
		ms, err := parseMillis(Option{Key: deadlineProp, Value: v})
		s.Optional.Deadline = ms
		return err
	},
	collectProp: func(s *api.AttributeSpec, v string) error {
		switch {
		case s.Required != nil:
			s.Required.Collect = api.InputCollect(v)
		case s.Optional != nil:
			s.Optional.Collect = api.InputCollect(v)
		default:
			return roleError(collectProp, api.RoleRequired)
		}
		return nil
	},
	forEachProp: func(s *api.AttributeSpec, v string) error {
		on, err := parseFlag(Option{Key: forEachProp, Value: v})
		if err != nil || !on {
			return err
		}
		// the Go field carries the element type, the attribute the array
		s.Type = api.TypeArray
		switch {
		case s.Required != nil:
			s.Required.ForEach = true
		case s.Optional != nil:
			s.Optional.ForEach = true
		default:
			return roleError(forEachProp, api.RoleRequired)
		}
		return nil
	},
	matchTag: func(s *api.AttributeSpec, v string) error {
		if s.Required == nil {
			return roleError(matchTag, api.RoleRequired)
		}
		cfg := &api.ScriptConfig{Language: api.ScriptLangJPath}
		if !parseScript(cfg, v) {
			return fmt.Errorf("%w: %q needs a script", ErrBadProp,
				matchTag)
		}
		s.Required.Match = cfg
		return nil
	},
	mappingProp: func(s *api.AttributeSpec, v string) error {
		mapping, err := initMapping(s, mappingProp)
		if err != nil {
			return err
		}
		mapping.Name = v
		return nil
	},
	mappingTag: func(s *api.AttributeSpec, v string) error {
		mapping, err := initMapping(s, mappingTag)
		if err != nil {
			return err
		}
		cfg := &api.ScriptConfig{Language: api.ScriptLangLua}
		if !parseScript(cfg, v) {
			return fmt.Errorf("%w: %q needs a script", ErrBadProp,
				mappingTag)
		}
		mapping.Script = cfg
		return nil
	},
}

func setRole(s *api.AttributeSpec, role api.AttributeRole) error {
	s.Required = nil
	s.Optional = nil
	s.Const = nil
	s.Meta = nil
	s.Output = nil
	switch role {
	case api.RoleRequired:
		s.Required = &api.RequiredConfig{}
	case api.RoleOptional:
		s.Optional = &api.OptionalConfig{}
	case api.RoleConst:
		s.Const = &api.ConstConfig{}
	case api.RoleMeta:
		s.Meta = &api.MetaConfig{}
	case api.RoleOutput:
		s.Output = &api.OutputConfig{}
	default:
		return fmt.Errorf("%w: unknown role %q", ErrBadProp, role)
	}
	s.Role = role
	return nil
}

// default and const values reach the engine as JSON
func jsonValue(t api.AttributeType, v string) string {
	if t == api.TypeString {
		return strconv.Quote(v)
	}
	return v
}

func roleError(prop string, role api.AttributeRole) error {
	return fmt.Errorf("%w: %q needs role %q", ErrBadProp, prop, role)
}

func parseFlag(o Option) (bool, error) {
	flag, err := strconv.ParseBool(o.Value)
	if err != nil {
		return false, fmt.Errorf("%w: %q needs true or false",
			ErrBadProp, o.Key)
	}
	return flag, nil
}

func parseMillis(o Option) (int64, error) {
	ms, err := strconv.ParseInt(o.Value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q needs milliseconds", ErrBadProp, o.Key)
	}
	return ms, nil
}

func parseInt(o Option) (int, error) {
	n, err := strconv.Atoi(o.Value)
	if err != nil {
		return 0, fmt.Errorf("%w: %q needs an integer", ErrBadProp, o.Key)
	}
	return n, nil
}

func initMapping(
	s *api.AttributeSpec, prop string,
) (*api.MappingConfig, error) {
	if mapping := s.Mapping(); mapping != nil {
		return mapping, nil
	}
	mapping := &api.MappingConfig{}
	switch {
	case s.Required != nil:
		s.Required.Mapping = mapping
	case s.Optional != nil:
		s.Optional.Mapping = mapping
	case s.Output != nil:
		s.Output.Mapping = mapping
	default:
		return nil, roleError(prop, api.RoleRequired)
	}
	return mapping, nil
}

func parseScript(cfg *api.ScriptConfig, value string) bool {
	cfg.Script = strings.TrimSpace(value)
	language, script, prefixed := strings.Cut(cfg.Script, optionAssign)
	language = strings.TrimSpace(language)
	known := language == api.ScriptLangLua || language == api.ScriptLangJPath
	if !prefixed || !known {
		return cfg.Script != ""
	}
	cfg.Language = language
	cfg.Script = strings.TrimSpace(script)
	return cfg.Script != ""
}

func applyOptions[T any](
	target *T, options Options, setters map[string]setter[T],
) error {
	for _, o := range options {
		set, ok := setters[o.Key]
		if !ok {
			return fmt.Errorf("%w: unknown property %q", ErrBadProp, o.Key)
		}
		if err := set(target, o.Value); err != nil {
			return err
		}
	}
	return nil
}
