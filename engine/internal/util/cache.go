package util

import (
	"container/list"
	"sync"
)

type (
	LRUCache[T any] struct {
		cache   map[string]*list.Element
		lru     *list.List
		maxSize int
		mu      sync.RWMutex
	}

	cacheEntry[T any] struct {
		value T
		key   string
	}
)

func NewLRUCache[T any](maxSize int) *LRUCache[T] {
	return &LRUCache[T]{
		cache:   map[string]*list.Element{},
		lru:     list.New(),
		maxSize: maxSize,
	}
}

func (c *LRUCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	elem, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		var zero T
		return zero, false
	}

	c.mu.Lock()
	c.lru.MoveToFront(elem)
	c.mu.Unlock()

	return elem.Value.(*cacheEntry[T]).value, true
}

func (c *LRUCache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry[T]).value = value
		return
	}

	entry := &cacheEntry[T]{key: key, value: value}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	if c.lru.Len() > c.maxSize {
		c.evictLast()
	}
}

func (c *LRUCache[T]) evictLast() {
	back := c.lru.Back()
	if back != nil {
		c.lru.Remove(back)
		backEntry := back.Value.(*cacheEntry[T])
		delete(c.cache, backEntry.key)
	}
}
