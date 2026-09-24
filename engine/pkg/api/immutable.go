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

// Set returns a copy of a map with the key set, creating it when nil
func Set[M ~map[K]V, K comparable, V any](m M, k K, v V) M {
	res := maps.Clone(m)
	if res == nil {
		res = make(M)
	}
	res[k] = v
	return res
}

// Delete returns a copy of a map without the key, or the map when absent
func Delete[M ~map[K]V, K comparable, V any](m M, k K) M {
	if _, ok := m[k]; !ok {
		return m
	}
	res := maps.Clone(m)
	delete(res, k)
	return res
}

// Apply returns a copy of a map with the other maps' entries applied
func Apply[M ~map[K]V, K comparable, V any](m M, others ...M) M {
	res := m
	copied := false
	for _, other := range others {
		if len(other) == 0 {
			continue
		}
		if !copied {
			res = maps.Clone(m)
			if res == nil {
				res = make(M)
			}
			copied = true
		}
		maps.Copy(res, other)
	}
	return res
}

// Append returns a copy of a slice with the values appended
func Append[S ~[]T, T any](s S, values ...T) S {
	res := make(S, len(s)+len(values))
	copy(res, s)
	copy(res[len(s):], values)
	return res
}

// Remove returns a copy of a slice without the values the predicate matches
func Remove[S ~[]T, T any](s S, fn func(T) bool) S {
	return slices.DeleteFunc(slices.Clone(s), fn)
}

// Set returns a copy of the Map with the key set, creating it when nil
func (m Map[K, V]) Set(k K, v V) Map[K, V] {
	return Set(m, k, v)
}

// Delete returns a copy of the Map without the key, or the Map when absent
func (m Map[K, V]) Delete(k K) Map[K, V] {
	return Delete(m, k)
}

// Apply returns a copy of the Map with the other Maps' entries applied
func (m Map[K, V]) Apply(others ...Map[K, V]) Map[K, V] {
	return Apply(m, others...)
}

// Append returns a copy of the Slice with the values appended
func (s Slice[T]) Append(values ...T) Slice[T] {
	return Append(s, values...)
}

// Remove returns a copy of the Slice without the values the predicate matches
func (s Slice[T]) Remove(fn func(T) bool) Slice[T] {
	return Remove(s, fn)
}
