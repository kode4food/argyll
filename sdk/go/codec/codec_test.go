package codec_test

import (
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"

	"github.com/kode4food/argyll/sdk/go/codec"
	"github.com/stretchr/testify/assert"
)

type (
	person struct {
		Name    string
		Age     int
		Tags    []string
		Nick    *string
		Ratings map[string]float64
		Active  bool
	}

	node struct {
		Name     string
		Children []node
	}

	failCodec struct{}
)

var (
	errCodec  = errors.New("codec failed")
	nodeImpl  codec.Codec[node]
	nodeCodec = codec.Ref(&nodeImpl)
)

func init() {
	nodeImpl = codec.Struct(
		codec.Field("name", codec.String, func(v *node) *string {
			return &v.Name
		}),
		codec.Field("children", codec.Slice(nodeCodec),
			func(v *node) *[]node {
				return &v.Children
			}),
	)
}

func personCodec() codec.Codec[person] {
	return codec.Struct(
		codec.Field("name", codec.String, func(v *person) *string {
			return &v.Name
		}),
		codec.Field("age", codec.Int, func(v *person) *int {
			return &v.Age
		}),
		codec.Field("tags", codec.Slice(codec.String),
			func(v *person) *[]string {
				return &v.Tags
			}),
		codec.Field("nick", codec.Optional(codec.String),
			func(v *person) **string {
				return &v.Nick
			}),
		codec.Field("ratings", codec.Map(codec.Float64),
			func(v *person) *map[string]float64 {
				return &v.Ratings
			}),
		codec.Field("active", codec.Bool, func(v *person) *bool {
			return &v.Active
		}),
	)
}

func TestScalars(t *testing.T) {
	s, err := codec.DecodeFrom(codec.String, strings.NewReader(`"hi"`))
	assert.NoError(t, err)
	assert.Equal(t, "hi", s)

	i, err := codec.DecodeFrom(codec.Int64,
		strings.NewReader("9007199254740993"))
	assert.NoError(t, err)
	assert.Equal(t, int64(9007199254740993), i)

	f, err := codec.DecodeFrom(codec.Float64, strings.NewReader("1.5"))
	assert.NoError(t, err)
	assert.Equal(t, 1.5, f)

	b, err := codec.DecodeFrom(codec.Bool, strings.NewReader("true"))
	assert.NoError(t, err)
	assert.True(t, b)
}

func TestScalarMismatch(t *testing.T) {
	_, err := codec.DecodeFrom(codec.String, strings.NewReader("12"))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)

	_, err = codec.DecodeFrom(codec.Int, strings.NewReader(`"x"`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)

	_, err = codec.DecodeFrom(codec.Bool, strings.NewReader(`"x"`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)
}

func TestEmptyInput(t *testing.T) {
	_, err := codec.DecodeFrom(codec.String, strings.NewReader(""))
	assert.ErrorIs(t, err, codec.ErrUnexpectedEnd)
}

func TestStructRoundTrip(t *testing.T) {
	c := personCodec()
	nick := "ace"
	in := person{
		Name:    "ada",
		Age:     36,
		Tags:    []string{"a", "b"},
		Nick:    &nick,
		Ratings: map[string]float64{"skill": 9.5},
		Active:  true,
	}

	var sb strings.Builder
	assert.NoError(t, codec.EncodeTo(c, &sb, in))

	out, err := codec.DecodeFrom(c, strings.NewReader(sb.String()))
	assert.NoError(t, err)
	assert.Equal(t, in.Name, out.Name)
	assert.Equal(t, in.Age, out.Age)
	assert.Equal(t, in.Tags, out.Tags)
	assert.Equal(t, nick, *out.Nick)
	assert.Equal(t, in.Ratings, out.Ratings)
	assert.True(t, out.Active)
}

func TestStructUnknownAndMissing(t *testing.T) {
	c := personCodec()
	src := `{"name":"ada","extra":{"deep":[1,2]},"nick":null}`

	out, err := codec.DecodeFrom(c, strings.NewReader(src))
	assert.NoError(t, err)
	assert.Equal(t, "ada", out.Name)
	assert.Equal(t, 0, out.Age)
	assert.Nil(t, out.Nick)
}

func TestStructFieldError(t *testing.T) {
	c := personCodec()
	_, err := codec.DecodeFrom(c, strings.NewReader(`{"age":"old"}`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)
	assert.Contains(t, err.Error(), `"age"`)
}

func TestNamedScalarTypes(t *testing.T) {
	type name string
	type count int32

	c := codec.Struct(
		codec.Field("name", codec.Text[name](),
			func(v *struct {
				Name  name
				Count count
			}) *name {
				return &v.Name
			}),
		codec.Field("count", codec.Number[count](),
			func(v *struct {
				Name  name
				Count count
			}) *count {
				return &v.Count
			}),
	)

	out, err := codec.DecodeFrom(c,
		strings.NewReader(`{"name":"x","count":3}`))
	assert.NoError(t, err)
	assert.Equal(t, name("x"), out.Name)
	assert.Equal(t, count(3), out.Count)
}

func TestNullComposites(t *testing.T) {
	out, err := codec.DecodeFrom(codec.Slice(codec.String),
		strings.NewReader("null"))
	assert.NoError(t, err)
	assert.Nil(t, out)

	m, err := codec.DecodeFrom(codec.Map(codec.String),
		strings.NewReader("null"))
	assert.NoError(t, err)
	assert.Nil(t, m)
}

func TestCompositeMismatch(t *testing.T) {
	_, err := codec.DecodeFrom(codec.Slice(codec.String),
		strings.NewReader(`{"a":1}`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)

	_, err = codec.DecodeFrom(codec.Slice(codec.String),
		strings.NewReader(`[1]`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)

	_, err = codec.DecodeFrom(codec.Map(codec.String),
		strings.NewReader(`[]`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)

	_, err = codec.DecodeFrom(personCodec(), strings.NewReader(`[]`))
	assert.ErrorIs(t, err, codec.ErrUnexpectedToken)
}

func TestOptionalEncoding(t *testing.T) {
	var sb strings.Builder
	assert.NoError(t, codec.EncodeTo(codec.Optional(codec.Int), &sb, nil))
	assert.Equal(t, "null", strings.TrimSpace(sb.String()))
}

func TestFloatEncoding(t *testing.T) {
	var sb strings.Builder
	assert.NoError(t, codec.EncodeTo(codec.Float64, &sb, 2.5))
	assert.Equal(t, "2.5", strings.TrimSpace(sb.String()))

	var ints strings.Builder
	assert.NoError(t, codec.EncodeTo(codec.Int, &ints, 7))
	assert.Equal(t, "7", strings.TrimSpace(ints.String()))
}

func TestRefRecursion(t *testing.T) {
	const src = `{"name":"a","children":[` +
		`{"name":"b","children":[{"name":"c","children":[]}]}]}`

	v, err := codec.DecodeFrom(nodeCodec, strings.NewReader(src))
	assert.NoError(t, err)
	assert.Equal(t, "c", v.Children[0].Children[0].Name)

	var out strings.Builder
	assert.NoError(t, codec.EncodeTo(nodeCodec, &out, v))
	assert.JSONEq(t, src, out.String())
}

func TestCompositeCodecErrors(t *testing.T) {
	fail := failCodec{}

	_, err := codec.DecodeFrom(codec.Slice[string](fail),
		strings.NewReader(`["x"]`))
	assert.ErrorIs(t, err, errCodec)

	_, err = codec.DecodeFrom(codec.Map[string](fail),
		strings.NewReader(`{"x":"y"}`))
	assert.ErrorIs(t, err, errCodec)

	var out strings.Builder
	assert.ErrorIs(t, codec.EncodeTo(codec.Slice[string](fail), &out,
		[]string{"x"}), errCodec)
	assert.ErrorIs(t, codec.EncodeTo(codec.Map[string](fail), &out,
		map[string]string{"x": "y"}), errCodec)
}

func (failCodec) Decode(*jsontext.Decoder) (string, error) {
	return "", errCodec
}

func (failCodec) Encode(*jsontext.Encoder, string) error {
	return errCodec
}
