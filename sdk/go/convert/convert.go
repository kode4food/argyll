// Package convert provides composable in-memory converters between Go values
// and the JSON-shaped values (string, bool, float64, []any, map[string]any, or
// nil) that decoding JSON into an any produces
package convert

import (
	"errors"
	"fmt"
	"math"
	"reflect"
)

type (
	// Converter turns a JSON-shaped value into a Go value and back
	Converter[T any] interface {
		From(any) (T, error)
		To(T) (any, error)
	}

	// StructField binds one object member to a field of struct S
	StructField[S any] interface {
		Name() string
		from(any, *S) error
		to(*S, *toState) (any, error)
	}

	// Numeric is any Go type whose JSON-shaped representation is a number
	Numeric interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
			~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
			~float32 | ~float64
	}

	stateConverter[T any] interface {
		to(T, *toState) (any, error)
	}

	textConverter[T ~string]   struct{}
	boolConverter[T ~bool]     struct{}
	numberConverter[T Numeric] struct{}
	sliceConverter[T any]      struct{ elem Converter[T] }
	optionalConverter[T any]   struct{ elem Converter[T] }
	mapConverter[T any]        struct{ elem Converter[T] }
	refConverter[T any]        struct{ target *Converter[T] }

	structConverter[S any] struct {
		fields []StructField[S]
	}

	structField[S, T any] struct {
		conv Converter[T]
		ptr  func(*S) *T
		name string
	}

	toState struct {
		active map[any]bool
	}
)

var (
	ErrUnexpectedValue = errors.New("unexpected value")
	ErrCyclicValue     = errors.New("cyclic value")
)

// Text returns a converter for any string-like type
func Text[T ~string]() Converter[T] {
	return textConverter[T]{}
}

// Boolean returns a converter for any bool-like type
func Boolean[T ~bool]() Converter[T] {
	return boolConverter[T]{}
}

// Number returns a converter for any numeric type
func Number[T Numeric]() Converter[T] {
	return numberConverter[T]{}
}

// Slice returns a converter for an array of the element converter's type
func Slice[T any](elem Converter[T]) Converter[[]T] {
	return sliceConverter[T]{elem: elem}
}

// Optional returns a converter mapping nil to a nil pointer
func Optional[T any](elem Converter[T]) Converter[*T] {
	return optionalConverter[T]{elem: elem}
}

// Map returns a converter for an object with uniformly typed members
func Map[T any](elem Converter[T]) Converter[map[string]T] {
	return mapConverter[T]{elem: elem}
}

// Ref returns a converter that reads target when used rather than when built,
// which lets a recursive type refer to its own converter
func Ref[T any](target *Converter[T]) Converter[T] {
	return refConverter[T]{target: target}
}

// Struct returns a converter for an object with statically known members
func Struct[S any](fields ...StructField[S]) Converter[S] {
	return &structConverter[S]{fields: fields}
}

// Field binds a named object member to a field of struct S. The ptr function
// returns the address of that field in a given struct value
func Field[S, T any](
	name string, c Converter[T], ptr func(*S) *T,
) StructField[S] {
	return structField[S, T]{name: name, conv: c, ptr: ptr}
}

// From accepts any string kind, so named string types such as identifiers read
// as text
func (textConverter[T]) From(v any) (T, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.String {
		return "", mismatch("string", v)
	}
	return T(r.String()), nil
}

func (textConverter[T]) To(v T) (any, error) {
	return string(v), nil
}

func (boolConverter[T]) From(v any) (T, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Bool {
		return false, mismatch("boolean", v)
	}
	return T(r.Bool()), nil
}

func (boolConverter[T]) To(v T) (any, error) {
	return bool(v), nil
}

// From accepts any numeric kind, and a fractional value for an integral type
// only when it is whole
func (numberConverter[T]) From(v any) (T, error) {
	r := reflect.ValueOf(v)
	switch {
	case r.CanInt():
		return T(r.Int()), nil
	case r.CanUint():
		return T(r.Uint()), nil
	case r.CanFloat():
		f := r.Float()
		if integral[T]() && f != math.Trunc(f) {
			return 0, mismatch("integer", v)
		}
		return T(f), nil
	default:
		return 0, mismatch("number", v)
	}
}

// To yields float64, the form a number takes once state is persisted and
// replayed
func (numberConverter[T]) To(v T) (any, error) {
	return float64(v), nil
}

func (c sliceConverter[T]) From(v any) ([]T, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, mismatch("array", v)
	}
	res := make([]T, len(items))
	for i, item := range items {
		elem, err := c.elem.From(item)
		if err != nil {
			return nil, fmt.Errorf("%w: [%d]", err, i)
		}
		res[i] = elem
	}
	return res, nil
}

func (c sliceConverter[T]) To(v []T) (any, error) {
	return valueOf(c, v, &toState{})
}

func (c optionalConverter[T]) From(v any) (*T, error) {
	if v == nil {
		return nil, nil
	}
	elem, err := c.elem.From(v)
	if err != nil {
		return nil, err
	}
	return &elem, nil
}

func (c optionalConverter[T]) To(v *T) (any, error) {
	return valueOf(c, v, &toState{})
}

func (c mapConverter[T]) From(v any) (map[string]T, error) {
	if v == nil {
		return nil, nil
	}
	members, ok := v.(map[string]any)
	if !ok {
		return nil, mismatch("object", v)
	}
	res := make(map[string]T, len(members))
	for name, member := range members {
		elem, err := c.elem.From(member)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", err, name)
		}
		res[name] = elem
	}
	return res, nil
}

func (c mapConverter[T]) To(v map[string]T) (any, error) {
	return valueOf(c, v, &toState{})
}

func (c refConverter[T]) From(v any) (T, error) {
	return (*c.target).From(v)
}

func (c refConverter[T]) To(v T) (any, error) {
	return valueOf(c, v, &toState{})
}

// From leaves absent members at their zero value and ignores unknown ones
func (c *structConverter[S]) From(v any) (S, error) {
	var res S
	if v == nil {
		return res, nil
	}
	members, ok := v.(map[string]any)
	if !ok {
		return res, mismatch("object", v)
	}
	for _, f := range c.fields {
		member, ok := members[f.Name()]
		if !ok {
			continue
		}
		if err := f.from(member, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (c *structConverter[S]) To(v S) (any, error) {
	return valueOf(c, v, &toState{})
}

func (f structField[S, T]) Name() string {
	return f.name
}

func (c sliceConverter[T]) to(v []T, state *toState) (any, error) {
	res := make([]any, len(v))
	for i, elem := range v {
		item, err := valueOf(c.elem, elem, state)
		if err != nil {
			return nil, err
		}
		res[i] = item
	}
	return res, nil
}

func (c optionalConverter[T]) to(v *T, state *toState) (any, error) {
	if v == nil {
		return nil, nil
	}
	if err := state.enter(v); err != nil {
		return nil, err
	}
	defer state.leave(v)
	return valueOf(c.elem, *v, state)
}

func (c mapConverter[T]) to(
	v map[string]T, state *toState,
) (any, error) {
	res := make(map[string]any, len(v))
	for name, elem := range v {
		member, err := valueOf(c.elem, elem, state)
		if err != nil {
			return nil, err
		}
		res[name] = member
	}
	return res, nil
}

func (c refConverter[T]) to(v T, state *toState) (any, error) {
	return valueOf(*c.target, v, state)
}

func (c *structConverter[S]) to(v S, state *toState) (any, error) {
	res := make(map[string]any, len(c.fields))
	for _, f := range c.fields {
		member, err := f.to(&v, state)
		if err != nil {
			return nil, err
		}
		res[f.Name()] = member
	}
	return res, nil
}

func (f structField[S, T]) from(v any, s *S) error {
	member, err := f.conv.From(v)
	if err != nil {
		return fmt.Errorf("%w: %q", err, f.name)
	}
	*f.ptr(s) = member
	return nil
}

func (f structField[S, T]) to(s *S, state *toState) (any, error) {
	return valueOf(f.conv, *f.ptr(s), state)
}

func (s *toState) enter(v any) error {
	if s.active[v] {
		return fmt.Errorf("%w: %T", ErrCyclicValue, v)
	}
	if s.active == nil {
		s.active = map[any]bool{}
	}
	s.active[v] = true
	return nil
}

func (s *toState) leave(v any) {
	delete(s.active, v)
}

func integral[T Numeric]() bool {
	return T(3)/T(2) == T(1)
}

func valueOf[T any](c Converter[T], v T, state *toState) (any, error) {
	if c, ok := c.(stateConverter[T]); ok {
		return c.to(v, state)
	}
	return c.To(v)
}

func mismatch(want string, got any) error {
	return fmt.Errorf("%w: expected %s, got %T", ErrUnexpectedValue, want, got)
}
