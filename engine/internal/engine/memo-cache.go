package engine

import (
	"errors"

	"github.com/kode4food/lru"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var ErrCacheMiss = errors.New("cache miss")

// MemoCache provides global caching of step results based on (step definition,
// inputs)
type MemoCache struct {
	cache *lru.Cache[api.Args]
}

// NewMemoCache creates a new memo cache with the specified maximum size
func NewMemoCache(maxSize int) *MemoCache {
	return &MemoCache{
		cache: lru.NewCache[api.Args](maxSize),
	}
}

// Get retrieves cached outputs for a step and inputs. Returns (outputs, true)
// on cache hit, (nil, false) on miss
func (m *MemoCache) Get(step *api.Step, inputs api.Args) (api.Args, bool) {
	key, err := buildCacheKey(step, inputs)
	if err != nil {
		return nil, false
	}

	result, err := m.cache.Get(key, func() (api.Args, error) {
		var zero api.Args
		return zero, ErrCacheMiss
	})
	if err != nil {
		return nil, false
	}

	return result, true
}

// Put stores cached outputs for a step and inputs
func (m *MemoCache) Put(
	step *api.Step, inputs api.Args, outputs api.Args,
) error {
	key, err := buildCacheKey(step, inputs)
	if err != nil {
		return err
	}

	_, err = m.cache.Get(key, func() (api.Args, error) {
		return outputs, nil
	})
	return err
}

// buildCacheKey creates a deterministic cache key from step definition and
// inputs. Format: stepKey + ":" + inputKey
func buildCacheKey(step *api.Step, inputs api.Args) (string, error) {
	stepKey, err := step.HashKey()
	if err != nil {
		return "", err
	}

	inputKey, err := inputs.HashKey()
	if err != nil {
		return "", err
	}

	return stepKey + ":" + inputKey, nil
}
