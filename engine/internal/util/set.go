package util

// Set is a generic set implementation for comparable values
type Set[K comparable] map[K]struct{}

// SetOf creates a new set containing the given elements
func SetOf[K comparable](elements ...K) Set[K] {
	s := make(Set[K], len(elements))
	for _, elem := range elements {
		s[elem] = struct{}{}
	}
	return s
}

// Add adds an element to the set
func (s Set[K]) Add(key K) {
	s[key] = struct{}{}
}

// Remove removes an element from the set
func (s Set[K]) Remove(key K) {
	delete(s, key)
}

// Contains returns true if the element exists in the set
func (s Set[K]) Contains(key K) bool {
	_, exists := s[key]
	return exists
}

// Len returns the number of elements in the set
func (s Set[K]) Len() int {
	return len(s)
}

// IsEmpty returns true if the set is empty
func (s Set[K]) IsEmpty() bool {
	return len(s) == 0
}
