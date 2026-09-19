package codec

import (
	"fmt"
	"math"
	"reflect"
)

// FromValue accepts any string kind, so named string types such as identifiers
// read as text
func (textCodec[T]) FromValue(v any) (T, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.String {
		return "", mismatch("string", v)
	}
	return T(r.String()), nil
}

func (textCodec[T]) ToValue(v T) (any, error) {
	return string(v), nil
}

func (boolCodec[T]) FromValue(v any) (T, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Bool {
		return false, mismatch("boolean", v)
	}
	return T(r.Bool()), nil
}

func (boolCodec[T]) ToValue(v T) (any, error) {
	return bool(v), nil
}

// FromValue accepts any numeric kind, but refuses a fractional value for an
// integral type rather than truncating it
func (numberCodec[T]) FromValue(v any) (T, error) {
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

// ToValue yields float64, the form a number takes once state is persisted and
// replayed
func (numberCodec[T]) ToValue(v T) (any, error) {
	return float64(v), nil
}

func (c sliceCodec[T]) FromValue(v any) ([]T, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, mismatch("array", v)
	}
	res := make([]T, len(items))
	for i, item := range items {
		elem, err := c.elem.FromValue(item)
		if err != nil {
			return nil, fmt.Errorf("%w: [%d]", err, i)
		}
		res[i] = elem
	}
	return res, nil
}

func (c sliceCodec[T]) ToValue(v []T) (any, error) {
	res := make([]any, len(v))
	for i, elem := range v {
		item, err := c.elem.ToValue(elem)
		if err != nil {
			return nil, err
		}
		res[i] = item
	}
	return res, nil
}

func (c stateSliceCodec[T]) ToValue(v []T) (any, error) {
	return valueOf(c, v, &encodeState{})
}

func (c optionalCodec[T]) FromValue(v any) (*T, error) {
	if v == nil {
		return nil, nil
	}
	elem, err := c.elem.FromValue(v)
	if err != nil {
		return nil, err
	}
	return &elem, nil
}

func (c optionalCodec[T]) ToValue(v *T) (any, error) {
	return valueOf(c, v, &encodeState{})
}

func (c mapCodec[T]) FromValue(v any) (map[string]T, error) {
	if v == nil {
		return nil, nil
	}
	members, ok := v.(map[string]any)
	if !ok {
		return nil, mismatch("object", v)
	}
	res := make(map[string]T, len(members))
	for name, member := range members {
		elem, err := c.elem.FromValue(member)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", err, name)
		}
		res[name] = elem
	}
	return res, nil
}

func (c mapCodec[T]) ToValue(v map[string]T) (any, error) {
	res := make(map[string]any, len(v))
	for name, elem := range v {
		member, err := c.elem.ToValue(elem)
		if err != nil {
			return nil, err
		}
		res[name] = member
	}
	return res, nil
}

func (c stateMapCodec[T]) ToValue(v map[string]T) (any, error) {
	return valueOf(c, v, &encodeState{})
}

func (c refCodec[T]) FromValue(v any) (T, error) {
	return (*c.target).FromValue(v)
}

func (c refCodec[T]) ToValue(v T) (any, error) {
	return valueOf(c, v, &encodeState{})
}

// FromValue leaves absent members at their zero value and ignores unknown ones,
// as Decode does
func (c *structCodec[S]) FromValue(v any) (S, error) {
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
		if err := f.fromValue(member, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (c *structCodec[S]) ToValue(v S) (any, error) {
	return valueOf(c, v, &encodeState{})
}

func (c stateSliceCodec[T]) toValue(
	v []T, state *encodeState,
) (any, error) {
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

func (c optionalCodec[T]) toValue(v *T, state *encodeState) (any, error) {
	if v == nil {
		return nil, nil
	}
	if err := state.enter(v); err != nil {
		return nil, err
	}
	defer state.leave(v)
	return valueOf(c.elem, *v, state)
}

func (c stateMapCodec[T]) toValue(
	v map[string]T, state *encodeState,
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

func (c refCodec[T]) toValue(v T, state *encodeState) (any, error) {
	return valueOf(*c.target, v, state)
}

func (c *structCodec[S]) toValue(v S, state *encodeState) (any, error) {
	res := make(map[string]any, len(c.fields))
	for _, f := range c.fields {
		member, err := f.toValue(&v, state)
		if err != nil {
			return nil, err
		}
		res[f.Name()] = member
	}
	return res, nil
}

func (f structField[S, T]) fromValue(v any, s *S) error {
	member, err := f.codec.FromValue(v)
	if err != nil {
		return fmt.Errorf("%w: %q", err, f.name)
	}
	*f.ptr(s) = member
	return nil
}

func (f structField[S, T]) toValue(s *S, state *encodeState) (any, error) {
	return valueOf(f.codec, *f.ptr(s), state)
}

func valueOf[T any](c Codec[T], v T, state *encodeState) (any, error) {
	if c, ok := c.(stateCodec[T]); ok {
		return c.toValue(v, state)
	}
	return c.ToValue(v)
}

func mismatch(want string, got any) error {
	return fmt.Errorf("%w: expected %s, got %T", ErrUnexpectedValue, want, got)
}
