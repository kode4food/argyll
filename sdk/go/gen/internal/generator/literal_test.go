package generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdk/go/gen/internal/generator"
)

func TestGoLiteralScalars(t *testing.T) {
	tests := map[string]struct {
		value any
		want  string
	}{
		"string":        {value: "a", want: `"a"`},
		"quoted string": {value: "a\"b\n", want: `"a\"b\n"`},
		"bool":          {value: true, want: "true"},
		"int":           {value: -3, want: "-3"},
		"int64":         {value: int64(1 << 40), want: "1099511627776"},
		"uint":          {value: uint(7), want: "7"},
		"uint8":         {value: uint8(255), want: "255"},
		"float64":       {value: 1.5, want: "1.5"},
		"float32":       {value: float32(0.25), want: "0.25"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := generator.GoLiteral(test.value)
			assert.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

// a named scalar renders as the bare constant the target type converts, the
// form a Go literal already infers wherever the value sits
func TestGoLiteralNamedScalars(t *testing.T) {
	sid, err := generator.GoLiteral(api.StepID("greet"))
	assert.NoError(t, err)
	assert.Equal(t, `"greet"`, sid)

	role, err := generator.GoLiteral(api.RoleRequired)
	assert.NoError(t, err)
	assert.Equal(t, `"required"`, role)
}

func TestGoLiteralStructSkipsZeroFields(t *testing.T) {
	got, err := generator.GoLiteral(api.HTTPAction{Endpoint: "/greet"})
	assert.NoError(t, err)
	assert.Equal(t, "api.HTTPAction{\nEndpoint: \"/greet\",\n}", got)
}

func TestGoLiteralEmptyComposite(t *testing.T) {
	got, err := generator.GoLiteral(api.HTTPAction{})
	assert.NoError(t, err)
	assert.Equal(t, "api.HTTPAction{}", got)
}

func TestGoLiteralPointer(t *testing.T) {
	got, err := generator.GoLiteral(&api.ScriptConfig{Language: "lua"})
	assert.NoError(t, err)
	assert.Equal(t, "&api.ScriptConfig{\nLanguage: \"lua\",\n}", got)

	var missing *api.ScriptConfig
	got, err = generator.GoLiteral(missing)
	assert.NoError(t, err)
	assert.Equal(t, "nil", got)
}

func TestGoLiteralSlice(t *testing.T) {
	got, err := generator.GoLiteral(api.Tags{"b", "a"})
	assert.NoError(t, err)
	assert.Equal(t, "api.Tags{\n\"b\",\n\"a\",\n}", got)
}

// a map renders its entries in a stable order, so regenerating a package that
// has not changed rewrites the same bytes
func TestGoLiteralMapIsOrdered(t *testing.T) {
	attrs := api.AttributeSpecs{
		"zeta":  {Role: api.RoleOutput},
		"alpha": {Role: api.RoleRequired},
		"mid":   {Role: api.RoleOptional},
	}

	first, err := generator.GoLiteral(attrs)
	assert.NoError(t, err)
	assert.Equal(t, "api.AttributeSpecs{\n"+
		"\"alpha\": {\nRole: \"required\",\n},\n"+
		"\"mid\": {\nRole: \"optional\",\n},\n"+
		"\"zeta\": {\nRole: \"output\",\n},\n}", first)

	for range 8 {
		again, err := generator.GoLiteral(attrs)
		assert.NoError(t, err)
		assert.Equal(t, first, again)
	}
}

func TestGoLiteralUnsupportedKind(t *testing.T) {
	_, err := generator.GoLiteral(make(chan int))
	assert.ErrorIs(t, err, generator.ErrUnsupportedGo)

	_, err = generator.GoLiteral(func() {})
	assert.ErrorIs(t, err, generator.ErrUnsupportedGo)

	_, err = generator.GoLiteral(nil)
	assert.ErrorIs(t, err, generator.ErrUnsupportedGo)
}

// an unnamed composite spells itself out of the types it composes, so a field
// like api.FlowConfig.Goals renders without a named type to reach for
func TestGoLiteralUnnamedComposite(t *testing.T) {
	tests := map[string]struct {
		value any
		want  string
	}{
		"slice": {
			value: []api.StepID{"greet"},
			want:  "[]api.StepID{\n\"greet\",\n}",
		},
		"array": {
			value: [2]string{"a", "b"},
			want:  "[2]string{\n\"a\",\n\"b\",\n}",
		},
		"map": {
			value: map[string]string{"a": "b"},
			want:  "map[string]string{\n\"a\": \"b\",\n}",
		},
		"pointer element": {
			value: []*api.ScriptConfig{{Language: "lua"}},
			want:  "[]*api.ScriptConfig{\n{\nLanguage: \"lua\",\n},\n}",
		},
		"nil pointer element": {
			value: []*api.ScriptConfig{nil},
			want:  "[]*api.ScriptConfig{\nnil,\n}",
		},
		"nested": {
			value: map[api.StepID][]string{"a": {"b"}},
			want:  "map[api.StepID][]string{\n\"a\": {\n\"b\",\n},\n}",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := generator.GoLiteral(test.value)
			assert.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGoLiteralUnnamedStruct(t *testing.T) {
	_, err := generator.GoLiteral(struct{ Name string }{Name: "a"})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)

	_, err = generator.GoLiteral([]struct{ Name string }{{Name: "a"}})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)

	_, err = generator.GoLiteral(map[string]struct{ Name string }{
		"a": {Name: "b"},
	})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)

	_, err = generator.GoLiteral(map[struct{ Name string }]string{
		{Name: "a"}: "b",
	})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)
}

func TestGoLiteralFlowGoals(t *testing.T) {
	got, err := generator.GoLiteral(api.Step{
		Flow: &api.FlowConfig{Goals: []api.StepID{"greet"}},
	})
	assert.NoError(t, err)
	assert.Contains(t, got, "Goals: []api.StepID{\n\"greet\",\n},")
}
