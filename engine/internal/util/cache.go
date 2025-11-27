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

	Constructor[T any] func() (T, error)

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

func (c *LRUCache[T]) Get(key string, create Constructor[T]) (T, error) {
	c.mu.RLock()
	elem, ok := c.cache[key]
	c.mu.RUnlock()

	if ok {
		c.mu.Lock()
		c.lru.MoveToFront(elem)
		c.mu.Unlock()
		return elem.Value.(*cacheEntry[T]).value, nil
	}

	value, err := create()
	if err != nil {
		var zero T
		return zero, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry[T]).value, nil
	}

	entry := &cacheEntry[T]{key: key, value: value}
	elem = c.lru.PushFront(entry)
	c.cache[key] = elem

	if c.lru.Len() > c.maxSize {
		c.evictLast()
	}

	return value, nil
}

func (c *LRUCache[T]) evictLast() {
	back := c.lru.Back()
	if back != nil {
		c.lru.Remove(back)
		backEntry := back.Value.(*cacheEntry[T])
		delete(c.cache, backEntry.key)
	}
}
