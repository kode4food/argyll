package api

import (
	"maps"
	"slices"
)

type (
	// Map is a map whose mutators return a modified copy, leaving the receiver
	// untouched
	Map[K comparable, V any] map[K]V

	// Slice is a slice whose mutators return a modified copy, leaving the
	// receiver's backing array untouched
	Slice[T any] []T
)

// Set returns a copy of the Map with the key set, creating it when nil
func (m Map[K, V]) Set(k K, v V) Map[K, V] {
	res := maps.Clone(m)
	if res == nil {
		res = Map[K, V]{}
	}
	res[k] = v
	return res
}

// Delete returns a copy of the Map without the key, or the Map when absent
func (m Map[K, V]) Delete(k K) Map[K, V] {
	if _, ok := m[k]; !ok {
		return m
	}
	res := maps.Clone(m)
	delete(res, k)
	return res
}

// Append returns a copy of the Slice with the values appended
func (s Slice[T]) Append(values ...T) Slice[T] {
	res := make(Slice[T], len(s)+len(values))
	copy(res, s)
	copy(res[len(s):], values)
	return res
}

// Remove returns a copy of the Slice without the values the predicate matches
func (s Slice[T]) Remove(fn func(T) bool) Slice[T] {
	return slices.DeleteFunc(slices.Clone(s), fn)
}
