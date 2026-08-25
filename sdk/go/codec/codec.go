// Package codec provides composable JSON codecs over encoding/json/jsontext
package codec

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
)

type (
	// Codec reads and writes a single Go value as a JSON value
	Codec[T any] interface {
		Decode(*jsontext.Decoder) (T, error)
		Encode(*jsontext.Encoder, T) error
	}

	// StructField binds one JSON object member to a field of struct S
	StructField[S any] interface {
		Name() string
		decode(*jsontext.Decoder, *S) error
		encode(*jsontext.Encoder, *S, *encodeState) error
	}

	// Numeric is any Go type whose JSON representation is a number
	Numeric interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
			~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
			~float32 | ~float64
	}

	stateCodec[T any] interface {
		encode(*jsontext.Encoder, T, *encodeState) error
	}

	textCodec[T ~string]   struct{}
	boolCodec[T ~bool]     struct{}
	numberCodec[T Numeric] struct{}
	stateSliceCodec[T any] struct{ sliceCodec[T] }
	stateMapCodec[T any]   struct{ mapCodec[T] }
	sliceCodec[T any]      struct{ elem Codec[T] }
	optionalCodec[T any]   struct{ elem Codec[T] }
	mapCodec[T any]        struct{ elem Codec[T] }
	refCodec[T any]        struct{ target *Codec[T] }

	structCodec[S any] struct {
		byName map[string]StructField[S]
		fields []StructField[S]
	}

	structField[S, T any] struct {
		codec Codec[T]
		ptr   func(*S) *T
		name  string
	}

	encodeState struct {
		active map[any]bool
	}
)

var (
	String  = Text[string]()
	Bool    = Boolean[bool]()
	Int     = Number[int]()
	Int64   = Number[int64]()
	Float64 = Number[float64]()

	ErrUnexpectedToken = errors.New("unexpected JSON token")
	ErrUnexpectedEnd   = errors.New("unexpected end of JSON input")
	ErrCyclicValue     = errors.New("cyclic value")
)

// Text returns a codec for any string-like type
func Text[T ~string]() Codec[T] {
	return textCodec[T]{}
}

// Boolean returns a codec for any bool-like type
func Boolean[T ~bool]() Codec[T] {
	return boolCodec[T]{}
}

// Number returns a codec for any numeric type
func Number[T Numeric]() Codec[T] {
	return numberCodec[T]{}
}

// Slice returns a codec for a JSON array of the element codec's type
func Slice[T any](elem Codec[T]) Codec[[]T] {
	c := sliceCodec[T]{elem: elem}
	if _, ok := elem.(stateCodec[T]); ok {
		return stateSliceCodec[T]{sliceCodec: c}
	}
	return c
}

// Optional returns a codec mapping JSON null to a nil pointer
func Optional[T any](elem Codec[T]) Codec[*T] {
	return optionalCodec[T]{elem: elem}
}

// Map returns a codec for a JSON object with uniformly typed members
func Map[T any](elem Codec[T]) Codec[map[string]T] {
	c := mapCodec[T]{elem: elem}
	if _, ok := elem.(stateCodec[T]); ok {
		return stateMapCodec[T]{mapCodec: c}
	}
	return c
}

// Ref returns a codec that reads target when used rather than when
// built, which lets a recursive type refer to its own codec
func Ref[T any](target *Codec[T]) Codec[T] {
	return refCodec[T]{target: target}
}

// Struct returns a codec for a JSON object with statically known members
func Struct[S any](fields ...StructField[S]) Codec[S] {
	byName := make(map[string]StructField[S], len(fields))
	for _, f := range fields {
		byName[f.Name()] = f
	}
	return &structCodec[S]{fields: fields, byName: byName}
}

// Field binds a named JSON object member to a field of struct S. The ptr
// function returns the address of that field in a given struct value
func Field[S, T any](
	name string, c Codec[T], ptr func(*S) *T,
) StructField[S] {
	return structField[S, T]{name: name, codec: c, ptr: ptr}
}

// DecodeFrom reads a single JSON value from r using the given codec
func DecodeFrom[T any](c Codec[T], r io.Reader) (T, error) {
	return c.Decode(jsontext.NewDecoder(r))
}

// EncodeTo writes v to w as a single JSON value using the given codec
func EncodeTo[T any](c Codec[T], w io.Writer, v T) error {
	e := jsontext.NewEncoder(w)
	if err := c.Encode(e, v); err != nil {
		return err
	}
	return nil
}

func (textCodec[T]) Decode(d *jsontext.Decoder) (T, error) {
	tok, err := readToken(d)
	if err != nil {
		return "", err
	}
	if tok.Kind() != '"' {
		return "", unexpected("string", tok.Kind())
	}
	return T(tok.String()), nil
}

func (textCodec[T]) Encode(e *jsontext.Encoder, v T) error {
	return e.WriteToken(jsontext.String(string(v)))
}

func (boolCodec[T]) Decode(d *jsontext.Decoder) (T, error) {
	tok, err := readToken(d)
	if err != nil {
		return false, err
	}
	if k := tok.Kind(); k != 't' && k != 'f' {
		return false, unexpected("boolean", tok.Kind())
	}
	return T(tok.Bool()), nil
}

func (boolCodec[T]) Encode(e *jsontext.Encoder, v T) error {
	return e.WriteToken(jsontext.Bool(bool(v)))
}

func (numberCodec[T]) Decode(d *jsontext.Decoder) (T, error) {
	tok, err := readToken(d)
	if err != nil {
		return 0, err
	}
	if tok.Kind() != '0' {
		return 0, unexpected("number", tok.Kind())
	}
	if integral[T]() {
		i, err := tok.Int()
		return T(i), err
	}
	f, err := tok.Float()
	return T(f), err
}

func (numberCodec[T]) Encode(e *jsontext.Encoder, v T) error {
	if integral[T]() {
		return e.WriteToken(jsontext.Int(int64(v)))
	}
	return e.WriteToken(jsontext.Float(float64(v)))
}

func (c sliceCodec[T]) Decode(d *jsontext.Decoder) ([]T, error) {
	if null, err := readNull(d); err != nil || null {
		return nil, err
	}
	if err := expect(d, '['); err != nil {
		return nil, err
	}
	res := []T{}
	for d.PeekKind() != ']' {
		v, err := c.elem.Decode(d)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, expect(d, ']')
}

func (c sliceCodec[T]) Encode(e *jsontext.Encoder, v []T) error {
	if err := e.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, elem := range v {
		if err := c.elem.Encode(e, elem); err != nil {
			return err
		}
	}
	return e.WriteToken(jsontext.EndArray)
}

func (c stateSliceCodec[T]) Encode(e *jsontext.Encoder, v []T) error {
	return encodeValue(c, e, v, &encodeState{})
}

func (c optionalCodec[T]) Decode(d *jsontext.Decoder) (*T, error) {
	if null, err := readNull(d); err != nil || null {
		return nil, err
	}
	v, err := c.elem.Decode(d)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c optionalCodec[T]) Encode(e *jsontext.Encoder, v *T) error {
	return encodeValue(c, e, v, &encodeState{})
}

func (c mapCodec[T]) Decode(d *jsontext.Decoder) (map[string]T, error) {
	if null, err := readNull(d); err != nil || null {
		return nil, err
	}
	if err := expect(d, '{'); err != nil {
		return nil, err
	}
	res := map[string]T{}
	for d.PeekKind() != '}' {
		name, err := readToken(d)
		if err != nil {
			return nil, err
		}
		v, err := c.elem.Decode(d)
		if err != nil {
			return nil, err
		}
		res[name.String()] = v
	}
	return res, expect(d, '}')
}

func (c mapCodec[T]) Encode(e *jsontext.Encoder, v map[string]T) error {
	if err := e.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for name, elem := range v {
		if err := e.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
		if err := c.elem.Encode(e, elem); err != nil {
			return err
		}
	}
	return e.WriteToken(jsontext.EndObject)
}

func (c stateMapCodec[T]) Encode(
	e *jsontext.Encoder, v map[string]T,
) error {
	return encodeValue(c, e, v, &encodeState{})
}

func (c refCodec[T]) Decode(d *jsontext.Decoder) (T, error) {
	return (*c.target).Decode(d)
}

func (c refCodec[T]) Encode(e *jsontext.Encoder, v T) error {
	return encodeValue(c, e, v, &encodeState{})
}

func (c *structCodec[S]) Decode(d *jsontext.Decoder) (S, error) {
	var res S
	if null, err := readNull(d); err != nil || null {
		return res, err
	}
	if err := expect(d, '{'); err != nil {
		return res, err
	}
	for d.PeekKind() != '}' {
		name, err := readToken(d)
		if err != nil {
			return res, err
		}
		f, ok := c.byName[name.String()]
		if !ok {
			if err := d.SkipValue(); err != nil {
				return res, err
			}
			continue
		}
		if err := f.decode(d, &res); err != nil {
			return res, err
		}
	}
	return res, expect(d, '}')
}

func (c *structCodec[S]) Encode(e *jsontext.Encoder, v S) error {
	return encodeValue(c, e, v, &encodeState{})
}

func (f structField[S, T]) Name() string {
	return f.name
}

func (c stateSliceCodec[T]) encode(
	e *jsontext.Encoder, v []T, state *encodeState,
) error {
	if err := e.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, elem := range v {
		if err := encodeValue(c.elem, e, elem, state); err != nil {
			return err
		}
	}
	return e.WriteToken(jsontext.EndArray)
}

func (c optionalCodec[T]) encode(
	e *jsontext.Encoder, v *T, state *encodeState,
) error {
	if v == nil {
		return e.WriteToken(jsontext.Null)
	}
	if err := state.enter(v); err != nil {
		return err
	}
	defer state.leave(v)
	return encodeValue(c.elem, e, *v, state)
}

func (c stateMapCodec[T]) encode(
	e *jsontext.Encoder, v map[string]T, state *encodeState,
) error {
	if err := e.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for name, elem := range v {
		if err := e.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
		if err := encodeValue(c.elem, e, elem, state); err != nil {
			return err
		}
	}
	return e.WriteToken(jsontext.EndObject)
}

func (c refCodec[T]) encode(
	e *jsontext.Encoder, v T, state *encodeState,
) error {
	return encodeValue(*c.target, e, v, state)
}

func (c *structCodec[S]) encode(
	e *jsontext.Encoder, v S, state *encodeState,
) error {
	if err := e.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, f := range c.fields {
		if err := e.WriteToken(jsontext.String(f.Name())); err != nil {
			return err
		}
		if err := f.encode(e, &v, state); err != nil {
			return err
		}
	}
	return e.WriteToken(jsontext.EndObject)
}

func (f structField[S, T]) decode(d *jsontext.Decoder, s *S) error {
	v, err := f.codec.Decode(d)
	if err != nil {
		return fmt.Errorf("%q: %w", f.name, err)
	}
	*f.ptr(s) = v
	return nil
}

func (f structField[S, T]) encode(
	e *jsontext.Encoder, s *S, state *encodeState,
) error {
	return encodeValue(f.codec, e, *f.ptr(s), state)
}

func (s *encodeState) enter(v any) error {
	if s.active[v] {
		return fmt.Errorf("%w: %T", ErrCyclicValue, v)
	}
	if s.active == nil {
		s.active = map[any]bool{}
	}
	s.active[v] = true
	return nil
}

func (s *encodeState) leave(v any) {
	delete(s.active, v)
}

func integral[T Numeric]() bool {
	return T(3)/T(2) == T(1)
}

func readNull(d *jsontext.Decoder) (bool, error) {
	if d.PeekKind() != 'n' {
		return false, nil
	}
	_, err := d.ReadToken()
	return true, err
}

func expect(d *jsontext.Decoder, kind jsontext.Kind) error {
	tok, err := readToken(d)
	if err != nil {
		return err
	}
	if tok.Kind() != kind {
		return unexpected(kind.String(), tok.Kind())
	}
	return nil
}

func readToken(d *jsontext.Decoder) (jsontext.Token, error) {
	tok, err := d.ReadToken()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return tok, ErrUnexpectedEnd
		}
		return tok, err
	}
	return tok.Clone(), nil
}

func unexpected(want string, got jsontext.Kind) error {
	return fmt.Errorf("%w: expected %s, got %s",
		ErrUnexpectedToken, want, got)
}

func encodeValue[T any](
	c Codec[T], e *jsontext.Encoder, v T, state *encodeState,
) error {
	if c, ok := c.(stateCodec[T]); ok {
		return c.encode(e, v, state)
	}
	return c.Encode(e, v)
}
