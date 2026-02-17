package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/jpath"
	"github.com/kode4food/lru"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const mappingCacheSize = 4096

var (
	ErrInvalidMapping = errors.New("invalid mapping")
)

var mappingCache = lru.NewCache[jpath.Path](mappingCacheSize)

func applyMapping(mapping string, value any) ([]any, error) {
	if mapping == "" {
		return nil, nil
	}

	path, err := compileMapping(mapping)
	if err != nil {
		return nil, err
	}

	return path(normalizeMappingDoc(value)), nil
}

func compileMapping(mapping string) (jpath.Path, error) {
	return mappingCache.Get(mapping, func() (jpath.Path, error) {
		parsed, err := jpath.Parse(mapping)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, mapping)
		}

		compiled, err := jpath.Compile(parsed)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, mapping)
		}
		return compiled, nil
	})
}

func mappingValue(mapping string, value any) (any, bool, error) {
	if mapping == "" {
		return value, true, nil
	}

	res, err := applyMapping(mapping, value)
	if err != nil {
		return nil, false, err
	}

	switch len(res) {
	case 0:
		return nil, false, nil
	case 1:
		return res[0], true, nil
	default:
		return res, true, nil
	}
}

func normalizeMappingDoc(value any) any {
	switch v := value.(type) {
	case api.Args:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = normalizeMappingDoc(elem)
		}
		return out
	case map[api.Name]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = normalizeMappingDoc(elem)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[key] = normalizeMappingDoc(elem)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for idx, elem := range v {
			out[idx] = normalizeMappingDoc(elem)
		}
		return out
	default:
		return value
	}
}
