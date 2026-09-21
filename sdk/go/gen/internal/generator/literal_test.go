package generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdk/go/gen/internal/generator"
)

func TestGoLiteral(t *testing.T) {
	// a named scalar renders as the bare constant its target type converts,
	// and an unnamed composite spells itself out of the types it composes
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
		"named string":  {value: api.StepID("greet"), want: `"greet"`},
		"named const":   {value: api.RoleRequired, want: `"required"`},
		"struct skips zero fields": {
			value: api.HTTPAction{Endpoint: "/greet"},
			want:  "api.HTTPAction{\nEndpoint: \"/greet\",\n}",
		},
		"empty composite": {
			value: api.HTTPAction{},
			want:  "api.HTTPAction{}",
		},
		"pointer": {
			value: &api.ScriptConfig{Language: "lua"},
			want:  "&api.ScriptConfig{\nLanguage: \"lua\",\n}",
		},
		"nil pointer": {
			value: (*api.ScriptConfig)(nil),
			want:  "nil",
		},
		"named slice": {
			value: api.Tags{"b", "a"},
			want:  "api.Tags{\n\"b\",\n\"a\",\n}",
		},
		"unnamed slice": {
			value: []api.StepID{"greet"},
			want:  "[]api.StepID{\n\"greet\",\n}",
		},
		"array": {
			value: [2]string{"a", "b"},
			want:  "[2]string{\n\"a\",\n\"b\",\n}",
		},
		"unnamed map": {
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
		"nested composite": {
			value: map[api.StepID][]string{"a": {"b"}},
			want:  "map[api.StepID][]string{\n\"a\": {\n\"b\",\n},\n}",
		},
		"step reaching an unnamed field type": {
			value: api.Step{
				Flow: &api.FlowConfig{Goals: []api.StepID{"greet"}},
			},
			want: "api.Step{\nFlow: &api.FlowConfig{\n" +
				"Goals: []api.StepID{\n\"greet\",\n},\n},\n}",
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

func TestGoLiteralRejects(t *testing.T) {
	tests := map[string]struct {
		value any
		want  error
	}{
		"channel":  {value: make(chan int), want: generator.ErrUnsupportedGo},
		"function": {value: func() {}, want: generator.ErrUnsupportedGo},
		"nil":      {value: nil, want: generator.ErrUnsupportedGo},
		"unnamed struct": {
			value: struct{ Name string }{Name: "a"},
			want:  generator.ErrUnnamedType,
		},
		"unnamed struct element": {
			value: []struct{ Name string }{{Name: "a"}},
			want:  generator.ErrUnnamedType,
		},
		"unnamed struct map value": {
			value: map[string]struct{ Name string }{"a": {Name: "b"}},
			want:  generator.ErrUnnamedType,
		},
		"unnamed struct map key": {
			value: map[struct{ Name string }]string{{Name: "a"}: "b"},
			want:  generator.ErrUnnamedType,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := generator.GoLiteral(test.value)
			assert.ErrorIs(t, err, test.want)
		})
	}
}

// a map renders its entries in a stable order, so regenerating a package that
// has not changed rewrites the same bytes
func TestGoLiteralMapOrder(t *testing.T) {
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
