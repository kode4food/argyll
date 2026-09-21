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

func TestGoLiteralNamedScalars(t *testing.T) {
	sid, err := generator.GoLiteral(api.StepID("greet"))
	assert.NoError(t, err)
	assert.Equal(t, `api.StepID("greet")`, sid)

	role, err := generator.GoLiteral(api.RoleRequired)
	assert.NoError(t, err)
	assert.Equal(t, `api.AttributeRole("required")`, role)
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
		"api.Name(\"alpha\"): &api.AttributeSpec{\n"+
		"Role: api.AttributeRole(\"required\"),\n},\n"+
		"api.Name(\"mid\"): &api.AttributeSpec{\n"+
		"Role: api.AttributeRole(\"optional\"),\n},\n"+
		"api.Name(\"zeta\"): &api.AttributeSpec{\n"+
		"Role: api.AttributeRole(\"output\"),\n},\n}", first)

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

func TestGoLiteralUnnamedType(t *testing.T) {
	_, err := generator.GoLiteral([]api.StepID{"greet"})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)

	_, err = generator.GoLiteral(struct{ Name string }{Name: "a"})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)

	_, err = generator.GoLiteral(map[string]string{"a": "b"})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)
}

func TestGoLiteralNestedError(t *testing.T) {
	_, err := generator.GoLiteral(api.Step{
		Attributes: api.AttributeSpecs{"a": {Role: api.RoleRequired}},
		Flow:       &api.FlowConfig{Goals: []api.StepID{"greet"}},
	})
	assert.ErrorIs(t, err, generator.ErrUnnamedType)
}
