package convert_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/sdk/go/convert"
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

	graph struct {
		Left     *graph
		Right    *graph
		Children []*graph
		Named    map[string]*graph
	}

	failConverter struct{}

	scalarName  string
	scalarCount int32
)

var errConvert = errors.New("convert failed")

func TestScalars(t *testing.T) {
	s, err := convert.Text[scalarName]().From(scalarName("x"))
	assert.NoError(t, err)
	assert.Equal(t, scalarName("x"), s)

	count := convert.Number[scalarCount]()
	i, err := count.From(3.0)
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(3), i)

	i, err = count.From(uint8(4))
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(4), i)

	i, err = count.From(int64(5))
	assert.NoError(t, err)
	assert.Equal(t, scalarCount(5), i)

	f, err := convert.Number[float64]().From(1.5)
	assert.NoError(t, err)
	assert.Equal(t, 1.5, f)

	b, err := convert.Boolean[bool]().From(true)
	assert.NoError(t, err)
	assert.True(t, b)

	v, err := convert.Number[int]().To(7)
	assert.NoError(t, err)
	assert.Equal(t, 7.0, v)

	v, err = convert.Boolean[bool]().To(true)
	assert.NoError(t, err)
	assert.Equal(t, true, v)
}

func TestScalarMismatch(t *testing.T) {
	_, err := convert.Text[string]().From(12.0)
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = convert.Number[int]().From("x")
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = convert.Number[int]().From(1.5)
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = convert.Boolean[bool]().From(nil)
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)
}

func TestStructRoundTrip(t *testing.T) {
	c := personConverter()
	nick := "ace"
	in := person{
		Name:    "ada",
		Age:     36,
		Tags:    []string{"a", "b"},
		Nick:    &nick,
		Ratings: map[string]float64{"skill": 9.5},
		Active:  true,
	}

	v, err := c.To(in)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"name":    "ada",
		"age":     36.0,
		"tags":    []any{"a", "b"},
		"nick":    "ace",
		"ratings": map[string]any{"skill": 9.5},
		"active":  true,
	}, v)

	out, err := c.From(v)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestStructUnknownAndMissing(t *testing.T) {
	out, err := personConverter().From(map[string]any{
		"name":  "ada",
		"extra": []any{1.0},
		"nick":  nil,
	})
	assert.NoError(t, err)
	assert.Equal(t, person{Name: "ada"}, out)

	out, err = personConverter().From(nil)
	assert.NoError(t, err)
	assert.Equal(t, person{}, out)
}

func TestComposites(t *testing.T) {
	nick := "ace"
	optional := convert.Optional(convert.Text[string]())
	v, err := optional.To(&nick)
	assert.NoError(t, err)
	assert.Equal(t, "ace", v)

	v, err = optional.To(nil)
	assert.NoError(t, err)
	assert.Nil(t, v)

	nodes := nodeConverter()
	v, err = convert.Slice(nodes).To([]node{{Name: "a"}})
	assert.NoError(t, err)
	assert.Equal(t,
		[]any{map[string]any{"name": "a", "children": []any{}}}, v)

	v, err = convert.Map(nodes).To(map[string]node{"a": {Name: "a"}})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"a": map[string]any{"name": "a", "children": []any{}},
	}, v)

	s, err := convert.Slice(convert.Number[int]()).From(nil)
	assert.NoError(t, err)
	assert.Nil(t, s)

	m, err := convert.Map(convert.Number[int]()).From(nil)
	assert.NoError(t, err)
	assert.Nil(t, m)
}

func TestCompositeMismatch(t *testing.T) {
	ints := convert.Slice(convert.Number[int]())
	intMap := convert.Map(convert.Number[int]())

	_, err := personConverter().From("ada")
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = personConverter().From(map[string]any{"age": "old"})
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)
	assert.Contains(t, err.Error(), `"age"`)

	_, err = ints.From([]any{1.0, "x"})
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)
	assert.Contains(t, err.Error(), "[1]")

	_, err = ints.From(map[string]any{})
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = intMap.From([]any{})
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)

	_, err = intMap.From(map[string]any{"a": "x"})
	assert.ErrorIs(t, err, convert.ErrUnexpectedValue)
}

func TestRefRecursion(t *testing.T) {
	nodes := nodeConverter()
	in := node{Name: "a", Children: []node{{Name: "b"}}}

	v, err := nodes.To(in)
	assert.NoError(t, err)

	out, err := nodes.From(v)
	assert.NoError(t, err)
	assert.Equal(t, "b", out.Children[0].Name)
}

func TestCycle(t *testing.T) {
	graphs := graphConverter()
	root := graph{}
	root.Left = &root
	_, err := graphs.To(root)
	assert.ErrorIs(t, err, convert.ErrCyclicValue)

	root = graph{}
	root.Children = []*graph{&root}
	_, err = graphs.To(root)
	assert.ErrorIs(t, err, convert.ErrCyclicValue)

	root = graph{}
	root.Named = map[string]*graph{"root": &root}
	_, err = graphs.To(root)
	assert.ErrorIs(t, err, convert.ErrCyclicValue)

	leaf := &graph{}
	_, err = graphs.To(graph{Left: leaf, Right: leaf})
	assert.NoError(t, err)
}

func TestCompositeErrors(t *testing.T) {
	fail := failConverter{}

	_, err := convert.Slice(fail).From([]any{"x"})
	assert.ErrorIs(t, err, errConvert)
	_, err = convert.Optional(fail).From("x")
	assert.ErrorIs(t, err, errConvert)

	_, err = convert.Slice(fail).To([]string{"x"})
	assert.ErrorIs(t, err, errConvert)
	_, err = convert.Map(fail).To(map[string]string{"x": "y"})
	assert.ErrorIs(t, err, errConvert)

	x := "x"
	_, err = convert.Map(convert.Optional(fail)).To(
		map[string]*string{"x": &x},
	)
	assert.ErrorIs(t, err, errConvert)
}

func (failConverter) From(any) (string, error) {
	return "", errConvert
}

func (failConverter) To(string) (any, error) {
	return nil, errConvert
}

func nodeConverter() convert.Converter[node] {
	var impl convert.Converter[node]
	res := convert.Ref(&impl)
	impl = convert.Struct(
		convert.Field("name", convert.Text[string](),
			func(v *node) *string {
				return &v.Name
			}),
		convert.Field("children", convert.Slice(res),
			func(v *node) *[]node {
				return &v.Children
			}),
	)
	return res
}

func graphConverter() convert.Converter[graph] {
	var impl convert.Converter[graph]
	res := convert.Ref(&impl)
	impl = convert.Struct(
		convert.Field("left", convert.Optional(res),
			func(v *graph) **graph {
				return &v.Left
			}),
		convert.Field("right", convert.Optional(res),
			func(v *graph) **graph {
				return &v.Right
			}),
		convert.Field("children",
			convert.Slice(convert.Optional(res)),
			func(v *graph) *[]*graph {
				return &v.Children
			}),
		convert.Field("named",
			convert.Map(convert.Optional(res)),
			func(v *graph) *map[string]*graph {
				return &v.Named
			}),
	)
	return res
}

func personConverter() convert.Converter[person] {
	return convert.Struct(
		convert.Field("name", convert.Text[string](),
			func(v *person) *string {
				return &v.Name
			}),
		convert.Field("age", convert.Number[int](), func(v *person) *int {
			return &v.Age
		}),
		convert.Field("tags", convert.Slice(convert.Text[string]()),
			func(v *person) *[]string {
				return &v.Tags
			}),
		convert.Field("nick", convert.Optional(convert.Text[string]()),
			func(v *person) **string {
				return &v.Nick
			}),
		convert.Field("ratings", convert.Map(convert.Number[float64]()),
			func(v *person) *map[string]float64 {
				return &v.Ratings
			}),
		convert.Field("active", convert.Boolean[bool](),
			func(v *person) *bool {
				return &v.Active
			}),
	)
}
