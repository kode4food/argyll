package util_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/util"
)

func TestCacheMiss(t *testing.T) {
	cache := util.NewLRUCache[string](10)
	callCount := 0

	value, err := cache.Get("key1", func() (string, error) {
		callCount++
		return "value1", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
	assert.Equal(t, 1, callCount)
}

func TestCacheHit(t *testing.T) {
	cache := util.NewLRUCache[string](10)
	callCount := 0

	cons := func() (string, error) {
		callCount++
		return "value1", nil
	}

	value1, err := cache.Get("key1", cons)
	assert.NoError(t, err)
	assert.Equal(t, "value1", value1)
	assert.Equal(t, 1, callCount)

	value2, err := cache.Get("key1", cons)
	assert.NoError(t, err)
	assert.Equal(t, "value1", value2)
	assert.Equal(t, 1, callCount)
}

func TestConstructorError(t *testing.T) {
	cache := util.NewLRUCache[string](10)
	expectedErr := errors.New("constructor error")

	value, err := cache.Get("key1", func() (string, error) {
		return "", expectedErr
	})

	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", value)
}

func TestEviction(t *testing.T) {
	cache := util.NewLRUCache[string](3)
	consCalls := make(map[string]int)

	cons := func(key string, value string) func() (string, error) {
		return func() (string, error) {
			consCalls[key]++
			return value, nil
		}
	}

	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		_, err := cache.Get(key, cons(key, value))
		assert.NoError(t, err)
	}

	_, err := cache.Get("key4", cons("key4", "value4"))
	assert.NoError(t, err)

	_, err = cache.Get("key1", cons("key1", "value1"))
	assert.NoError(t, err)
	assert.Equal(t, 2, consCalls["key1"])
}

func TestLRUOrdering(t *testing.T) {
	cache := util.NewLRUCache[string](3)
	consCalls := make(map[string]int)

	cons := func(key string) func() (string, error) {
		return func() (string, error) {
			consCalls[key]++
			return key, nil
		}
	}

	_, _ = cache.Get("key1", cons("key1"))
	_, _ = cache.Get("key2", cons("key2"))
	_, _ = cache.Get("key3", cons("key3"))

	_, _ = cache.Get("key1", cons("key1"))

	_, _ = cache.Get("key4", cons("key4"))

	_, _ = cache.Get("key2", cons("key2"))

	assert.Equal(t, 2, consCalls["key2"])
}
