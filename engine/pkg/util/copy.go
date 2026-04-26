package util

// MutableCopy returns a new pointer to a shallow copy of val.
//
// Use MutableCopy when code needs to derive a new mutable value from an
// existing persistent value before applying changes. If val is nil, MutableCopy
// returns a pointer to the zero value of T
func MutableCopy[T any](val *T) *T {
	if val == nil {
		var zero T
		return &zero
	}
	res := *val
	return &res
}
