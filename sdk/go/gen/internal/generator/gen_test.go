package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kode4food/argyll/sdk/go/gen/internal/generator"
	"github.com/stretchr/testify/assert"
)

func render(t *testing.T, pattern string) ([]byte, error) {
	t.Helper()
	pkgs, err := generator.Load(".", pattern)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	return generator.Render(pkgs[0])
}

func TestNames(t *testing.T) {
	assert.Equal(t, "customer_id", generator.SnakeCase("CustomerID"))
	assert.Equal(t, "http_server", generator.SnakeCase("HTTPServer"))
	assert.Equal(t, "score", generator.SnakeCase("Score"))
	assert.Equal(t, "calculate-risk", generator.KebabCase("CalculateRisk"))
	assert.Equal(t, "Calculate Risk", generator.TitleCase("CalculateRisk"))
	assert.Equal(t, "CustomerId", generator.ExportedName("customer-id"))
	assert.Equal(t, "Score", generator.ExportedName("score"))
}

func TestGeneratedFileMatchesRender(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)

	path := filepath.Join("..", "..", "..", "example",
		generator.GeneratedFile)
	built, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, string(built), string(src))
}

func TestContractInference(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, `ID:   "calculate-risk"`)
	assert.Contains(t, text, `Name: "Calculate Risk"`)
	assert.Contains(t, text, `{Name: "customer_id", Type: api.TypeString}`)
	assert.Contains(t, text, `{Name: "amount", Type: api.TypeNumber}`)
	assert.Contains(t, text, `{Name: "tags", Type: api.TypeArray}`)
	assert.Contains(t, text,
		`{Name: "note", Type: api.TypeString, Optional: true}`)
	assert.Contains(t, text, `{Name: "approved", Type: api.TypeBoolean}`)

	// an error is control-plane information, never a step output
	assert.NotContains(t, text, `Name: "err`)
}

func TestWrapContract(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, `ID:   "score-customer"`)
	assert.Contains(t, text, `{Name: "customer-id", Type: api.TypeString}`)
	assert.Contains(t, text,
		"r0, r1, err := ScoreCustomer(in.CustomerId, in.Amount)")
	assert.Contains(t, text,
		"return argyllScoreCustomerOut{Score: r0, Approved: r1}, nil")
}

func TestZeroOutputStep(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, `ID:   "audit"`)
	assert.Contains(t, text, "codec.Struct[struct{}]()")
	assert.Contains(t, text, "return struct{}{}, Audit(in)")
}

func TestCompositeCodecs(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, "codec.Map(codec.Number[int]())")
	assert.Contains(t, text, "codec.Slice(codec.Text[string]())")
	assert.Contains(t, text, "codec.Optional(codec.Text[string]())")
	assert.Contains(t, text, `{Name: "address", Type: api.TypeObject}`)
	assert.Contains(t, text, `{Name: "limits", Type: api.TypeObject}`)
}

func TestFieldTags(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, `{Name: "iso_currency", Type: api.TypeString}`)
	assert.Contains(t, text, `codec.Field("iso_currency"`)

	// a tag of "-" keeps the field off the wire entirely
	assert.NotContains(t, text, "scratch")
	assert.NotContains(t, text, "Scratch")
}

func TestRecursiveCodec(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	// a self-referential codec cannot be a plain var initializer
	assert.Contains(t, text, "var argyllCodecNodeImpl codec.Codec[Node]")
	assert.Contains(t, text,
		"var argyllCodecNode = codec.Ref(&argyllCodecNodeImpl)")
	assert.Contains(t, text, "argyllCodecNodeImpl = codec.Struct(")
	assert.Contains(t, text, "codec.Slice(argyllCodecNode)")
	assert.Contains(t, text, "codec.Optional(argyllCodecNode)")

	// non-recursive codecs stay plain
	assert.Contains(t, text, "var argyllCodecRiskArgs = codec.Struct(")
}

func TestNoDirectives(t *testing.T) {
	src, err := render(t, "./testdata/nodirectives")
	assert.NoError(t, err)
	assert.Nil(t, src)
}

func TestDiagnostics(t *testing.T) {
	tests := map[string]struct {
		pattern string
		wants   []string
	}{
		"step arity": {
			pattern: "./testdata/badstep",
			wants:   []string{"TooManyArgs", "one arguments struct"},
		},
		"wrap arity": {
			pattern: "./testdata/badwrap",
			wants:   []string{"Add", "declares 2 outputs but returns 1"},
		},
		"label syntax": {
			pattern: "./testdata/badlabel",
			wants:   []string{`label "domain" is not key=value`},
		},
		"attribute type": {
			pattern: "./testdata/badtype",
			wants:   []string{"chan int"},
		},
		"named composite": {
			pattern: "./testdata/badnamed",
			wants:   []string{"unsupported attribute type: Tags"},
		},
		"map key type": {
			pattern: "./testdata/badmapkey",
			wants:   []string{"int keys"},
		},
		"tag option": {
			pattern: "./testdata/badtag",
			wants:   []string{`unknown option "omitempty" on Amount`},
		},
		"tag name": {
			pattern: "./testdata/badtagname",
			wants:   []string{`bad attribute name "order amount"`},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := render(t, tt.pattern)
			if !assert.Error(t, err) {
				return
			}
			for _, want := range tt.wants {
				assert.Contains(t, err.Error(), want)
			}
			assert.Contains(t, err.Error(), "steps.go")
		})
	}
}

func TestGenerateIsIdempotent(t *testing.T) {
	_, err := generator.Generate(".", "../../../example")
	assert.NoError(t, err)

	written, err := generator.Generate(".", "../../../example")
	assert.NoError(t, err)
	assert.Empty(t, written)
}

func TestGenerateRemovesStaleFile(t *testing.T) {
	dir := filepath.Join("testdata", "nodirectives")
	path := filepath.Join(dir, generator.GeneratedFile)
	assert.NoError(t, os.WriteFile(path, []byte("package nodirectives\n"),
		generator.FileMode))

	_, err := generator.Generate(".", "./"+dir)
	assert.NoError(t, err)
	assert.NoFileExists(t, path)
}

func TestLabels(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, "Labels: api.Labels{")
	assert.Contains(t, text, `"description": "score a customer for risk"`)
	assert.Contains(t, text, `"domain":      "risk"`)

	// a step without label directives declares no labels
	assert.NotContains(t, text, "Labels: api.Labels{}")
}
