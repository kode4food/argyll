package generator

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// stepSetter applies one declared property to a step
	stepSetter func(*api.Step, string) error

	// attrSetter applies one declared property to an attribute
	attrSetter func(*api.AttributeSpec, string) error
)

const (
	nameProp      = "name"
	memoizeProp   = "memoize"
	timeoutProp   = "timeout"
	predicateProp = "predicate"

	roleProp     = "role"
	defaultProp  = "default"
	valueProp    = "value"
	keyProp      = "key"
	collectProp  = "collect"
	deadlineProp = "deadline"
	forEachProp  = "for_each"
	matchProp    = "match"
	mappingProp  = "mapping"

	// scripts declared in a directive or a tag are Lua
	scriptLanguage = "lua"
)

var (
	ErrBadProp = errors.New("invalid argyll property")
)

var stepSetters = map[string]stepSetter{
	nameProp: func(s *api.Step, v string) error {
		s.Name = api.Name(v)
		return nil
	},
	memoizeProp: func(s *api.Step, v string) error {
		on, err := parseFlag(Option{Key: memoizeProp, Value: v})
		s.Memoizable = on
		return err
	},
	timeoutProp: func(s *api.Step, v string) error {
		ms, err := parseMillis(Option{Key: timeoutProp, Value: v})
		s.HTTP.Invoke.Timeout = ms
		return err
	},
	predicateProp: func(s *api.Step, v string) error {
		s.Predicate = &api.ScriptConfig{Language: scriptLanguage, Script: v}
		return nil
	},
}

// applied after the role, so each setter finds the config it writes into
var attrSetters = map[string]attrSetter{
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
	matchProp: func(s *api.AttributeSpec, v string) error {
		if s.Required == nil {
			return roleError(matchProp, api.RoleRequired)
		}
		s.Required.Match = &api.ScriptConfig{
			Language: scriptLanguage, Script: v,
		}
		return nil
	},
	mappingProp: func(s *api.AttributeSpec, v string) error {
		mapping := &api.MappingConfig{Name: v}
		switch {
		case s.Required != nil:
			s.Required.Mapping = mapping
		case s.Optional != nil:
			s.Optional.Mapping = mapping
		case s.Output != nil:
			s.Output.Mapping = mapping
		default:
			return roleError(mappingProp, api.RoleRequired)
		}
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
