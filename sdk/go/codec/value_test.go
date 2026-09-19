package codec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/sdk/go/codec"
)

func TestValueScalars(t *testing.T) {
	s, err := codec.Text[scalarName]().FromValue(scalarName("x"))
	assert.NoError(t, err)
	assert.Equal(t, scalarName("x"), s)

	i, err := codec.Number[scalarCount]().FromValue(3.0)
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(3), i)

	i, err = codec.Number[scalarCount]().FromValue(uint8(4))
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(4), i)

	i, err = codec.Number[scalarCount]().FromValue(int64(5))
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(5), i)

	f, err := codec.Float64.FromValue(1.5)
	assert.NoError(t, err)
	assert.Equal(t, 1.5, f)

	b, err := codec.Bool.FromValue(true)
	assert.NoError(t, err)
	assert.True(t, b)

	v, err := codec.Int.ToValue(7)
	assert.NoError(t, err)
	assert.Equal(t, 7.0, v)
}

func TestValueScalarMismatch(t *testing.T) {
	_, err := codec.String.FromValue(12.0)
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = codec.Int.FromValue("x")
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = codec.Int.FromValue(1.5)
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = codec.Bool.FromValue(nil)
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)
}

func TestValueStructRoundTrip(t *testing.T) {
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

	v, err := c.ToValue(in)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"name":    "ada",
		"age":     36.0,
		"tags":    []any{"a", "b"},
		"nick":    "ace",
		"ratings": map[string]any{"skill": 9.5},
		"active":  true,
	}, v)

	out, err := c.FromValue(v)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestValueStructUnknownAndMissing(t *testing.T) {
	out, err := personCodec().FromValue(map[string]any{
		"name":  "ada",
		"extra": []any{1.0},
		"nick":  nil,
	})
	assert.NoError(t, err)
	assert.Equal(t, person{Name: "ada"}, out)

	out, err = personCodec().FromValue(nil)
	assert.NoError(t, err)
	assert.Equal(t, person{}, out)
}

func TestValueComposites(t *testing.T) {
	nick := "ace"
	v, err := codec.Optional(codec.String).ToValue(&nick)
	assert.NoError(t, err)
	assert.Equal(t, "ace", v)

	v, err = codec.Optional(codec.String).ToValue(nil)
	assert.NoError(t, err)
	assert.Nil(t, v)

	v, err = codec.Slice(nodeCodec).ToValue([]node{{Name: "a"}})
	assert.NoError(t, err)
	assert.Equal(t,
		[]any{map[string]any{"name": "a", "children": []any{}}}, v)

	v, err = codec.Map(nodeCodec).ToValue(map[string]node{"a": {Name: "a"}})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"a": map[string]any{"name": "a", "children": []any{}},
	}, v)

	s, err := codec.Slice(codec.Int).FromValue(nil)
	assert.NoError(t, err)
	assert.Nil(t, s)

	m, err := codec.Map(codec.Int).FromValue(nil)
	assert.NoError(t, err)
	assert.Nil(t, m)
}

func TestValueCompositeMismatch(t *testing.T) {
	_, err := personCodec().FromValue("ada")
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = personCodec().FromValue(map[string]any{"age": "old"})
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)
	assert.Contains(t, err.Error(), `"age"`)

	_, err = codec.Slice(codec.Int).FromValue([]any{1.0, "x"})
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)
	assert.Contains(t, err.Error(), "[1]")

	_, err = codec.Slice(codec.Int).FromValue(map[string]any{})
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = codec.Map(codec.Int).FromValue([]any{})
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)

	_, err = codec.Map(codec.Int).FromValue(map[string]any{"a": "x"})
	assert.ErrorIs(t, err, codec.ErrUnexpectedValue)
}

func TestValueRefRecursion(t *testing.T) {
	in := node{Name: "a", Children: []node{{Name: "b"}}}

	v, err := nodeCodec.ToValue(in)
	assert.NoError(t, err)

	out, err := nodeCodec.FromValue(v)
	assert.NoError(t, err)
	assert.Equal(t, "b", out.Children[0].Name)
}

func TestValueCycle(t *testing.T) {
	root := graph{}
	root.Left = &root
	_, err := graphCodec.ToValue(root)
	assert.ErrorIs(t, err, codec.ErrCyclicValue)

	root = graph{}
	root.Children = []*graph{&root}
	_, err = graphCodec.ToValue(root)
	assert.ErrorIs(t, err, codec.ErrCyclicValue)

	root = graph{}
	root.Named = map[string]*graph{"root": &root}
	_, err = graphCodec.ToValue(root)
	assert.ErrorIs(t, err, codec.ErrCyclicValue)

	leaf := &graph{}
	_, err = graphCodec.ToValue(graph{Left: leaf, Right: leaf})
	assert.NoError(t, err)
}

func TestValueCompositeErrors(t *testing.T) {
	fail := failCodec{}

	_, err := codec.Slice(fail).FromValue([]any{"x"})
	assert.ErrorIs(t, err, errCodec)
	_, err = codec.Optional(fail).FromValue("x")
	assert.ErrorIs(t, err, errCodec)

	_, err = codec.Slice(fail).ToValue([]string{"x"})
	assert.ErrorIs(t, err, errCodec)
	_, err = codec.Map(fail).ToValue(map[string]string{"x": "y"})
	assert.ErrorIs(t, err, errCodec)

	x := "x"
	_, err = codec.Map(codec.Optional(fail)).ToValue(
		map[string]*string{"x": &x},
	)
	assert.ErrorIs(t, err, errCodec)
}
