package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLRUCache(t *testing.T) {
	cache := NewLRUCache[string](10)

	require.NotNil(t, cache)
	assert.Equal(t, 10, cache.maxSize)
	assert.NotNil(t, cache.cache)
	assert.NotNil(t, cache.lru)
}

func TestCacheMiss(t *testing.T) {
	cache := NewLRUCache[string](10)
	callCount := 0

	value, err := cache.Get("key1", func() (string, error) {
		callCount++
		return "value1", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "value1", value)
	assert.Equal(t, 1, callCount)
}

func TestCacheHit(t *testing.T) {
	cache := NewLRUCache[string](10)
	callCount := 0

	cons := func() (string, error) {
		callCount++
		return "value1", nil
	}

	value1, err := cache.Get("key1", cons)
	require.NoError(t, err)
	assert.Equal(t, "value1", value1)
	assert.Equal(t, 1, callCount)

	value2, err := cache.Get("key1", cons)
	require.NoError(t, err)
	assert.Equal(t, "value1", value2)
	assert.Equal(t, 1, callCount)
}

func TestConstructorError(t *testing.T) {
	cache := NewLRUCache[string](10)
	expectedErr := errors.New("constructor error")

	value, err := cache.Get("key1", func() (string, error) {
		return "", expectedErr
	})

	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", value)
}

func TestEviction(t *testing.T) {
	cache := NewLRUCache[string](3)
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
		require.NoError(t, err)
	}

	assert.Equal(t, 3, cache.lru.Len())

	_, err := cache.Get("key4", cons("key4", "value4"))
	require.NoError(t, err)
	assert.Equal(t, 3, cache.lru.Len())

	_, err = cache.Get("key1", cons("key1", "value1"))
	require.NoError(t, err)
	assert.Equal(t, 2, consCalls["key1"])
}

func TestLRUOrdering(t *testing.T) {
	cache := NewLRUCache[string](3)
	consCalls := make(map[string]int)

	cons := func(key string) func() (string, error) {
		return func() (string, error) {
			consCalls[key]++
			return key, nil
		}
	}

	cache.Get("key1", cons("key1"))
	cache.Get("key2", cons("key2"))
	cache.Get("key3", cons("key3"))

	cache.Get("key1", cons("key1"))

	cache.Get("key4", cons("key4"))

	cache.Get("key2", cons("key2"))

	assert.Equal(t, 2, consCalls["key2"])
}
