package memo

import (
	"errors"

	"github.com/kode4food/lru"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Cache provides caching of step results based on step definition and inputs
type Cache struct {
	cache *lru.Cache[api.Args]
}

var ErrCacheMiss = errors.New("cache miss")

// NewCache creates a memo cache with the specified maximum size
func NewCache(maxSize int) *Cache {
	return &Cache{
		cache: lru.NewCache[api.Args](maxSize),
	}
}

// Get retrieves cached outputs for a step and inputs
func (c *Cache) Get(step *api.Step, inputs api.Args) (api.Args, bool) {
	key, err := buildCacheKey(step, inputs)
	if err != nil {
		return nil, false
	}

	result, err := c.cache.Get(key, func() (api.Args, error) {
		var zero api.Args
		return zero, ErrCacheMiss
	})
	if err != nil {
		return nil, false
	}

	return result, true
}

// Put stores cached outputs for a step and inputs
func (c *Cache) Put(step *api.Step, inputs api.Args, outputs api.Args) error {
	key, err := buildCacheKey(step, inputs)
	if err != nil {
		return err
	}

	_, err = c.cache.Get(key, func() (api.Args, error) {
		return outputs, nil
	})
	return err
}

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
