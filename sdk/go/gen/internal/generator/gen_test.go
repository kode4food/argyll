package generator_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/sdk/go/gen/internal/generator"
	"github.com/stretchr/testify/assert"
)

var specPattern = regexp.MustCompile(`(?m)^\tSpec:\s+(".*"),$`)

func TestNames(t *testing.T) {
	assert.Equal(t, "customer_id", generator.SnakeCase("CustomerID"))
	assert.Equal(t, "http_server", generator.SnakeCase("HTTPServer"))
	assert.Equal(t, "score", generator.SnakeCase("Score"))
	assert.Equal(t, "calculate-risk", generator.KebabCase("CalculateRisk"))
	assert.Equal(t, "Calculate Risk", generator.TitleCase("CalculateRisk"))
	assert.Equal(t, "CustomerId", generator.ExportedName("customer-id"))
	assert.Equal(t, "Score", generator.ExportedName("score"))
}

func TestParseOptions(t *testing.T) {
	t.Run("head only", func(t *testing.T) {
		head, opts, err := generator.ParseOptions("  iso_currency  ")
		assert.NoError(t, err)
		assert.Equal(t, "iso_currency", head)
		assert.Empty(t, opts)
	})

	t.Run("options only", func(t *testing.T) {
		head, opts, err := generator.ParseOptions(" optional:true ")
		assert.NoError(t, err)
		assert.Empty(t, head)
		assert.Equal(t, generator.Options{
			{Key: "optional", Value: "true"},
		}, opts)
	})

	t.Run("head and options, loosely spaced", func(t *testing.T) {
		head, opts, err := generator.ParseOptions(
			" flow ; meta : flow_id ; label : domain = risk ")
		assert.NoError(t, err)
		assert.Equal(t, "flow", head)
		assert.Equal(t, generator.Options{
			{Key: "meta", Value: "flow_id"},
			{Key: "label", Value: "domain = risk"},
		}, opts)
	})

	t.Run("empty segments are skipped", func(t *testing.T) {
		head, opts, err := generator.ParseOptions("charge-card;;name:Charge;")
		assert.NoError(t, err)
		assert.Equal(t, "charge-card", head)
		assert.Len(t, opts, 1)
	})

	t.Run("an option needs a value", func(t *testing.T) {
		_, _, err := generator.ParseOptions("charge-card;name")
		assert.True(t, errors.Is(err, generator.ErrBadOption))
	})
}

func TestSplitHead(t *testing.T) {
	tests := map[string]generator.Head{
		"score-customer":         {Name: "score-customer"},
		" (a, b) -> (c) ":        {Attrs: "(a, b) -> (c)"},
		"score-v2 (a, b) -> (c)": {Name: "score-v2", Attrs: "(a, b) -> (c)"},
		"score-v2 -> (c)":        {Name: "score-v2", Attrs: "-> (c)"},
		"multi-word-id":          {Name: "multi-word-id"},
	}

	for text, want := range tests {
		t.Run(text, func(t *testing.T) {
			assert.Equal(t, want, generator.SplitHead(text))
		})
	}
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
	step := steps(t, "../../../example")["calculate-risk"]
	assert.Equal(t, api.Name("Calculate Risk"), step.Name)
	assert.Equal(t, api.StepTypeSync, step.Type)

	attrs := step.Attributes
	assert.Equal(t, api.TypeString, attrs["customer_id"].Type)
	assert.Equal(t, api.TypeNumber, attrs["amount"].Type)
	assert.Equal(t, api.TypeArray, attrs["tags"].Type)
	assert.Equal(t, api.TypeBoolean, attrs["approved"].Type)
	assert.True(t, attrs["customer_id"].IsRequired())
	assert.True(t, attrs["approved"].IsOutput())

	// a pointer field is an optional attribute
	assert.True(t, attrs["note"].IsOptional())

	// an error is control-plane information, never a step output
	assert.NotContains(t, attrs, api.Name("err"))
}

func TestWrapContract(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	step := steps(t, "../../../example")["score-customer"]
	assert.Equal(t, api.TypeString, step.Attributes["customer-id"].Type)
	assert.Contains(t, text,
		"r0, r1, err := ScoreCustomer(in.CustomerId, in.Amount)")
	assert.Contains(t, text,
		"return argyllScoreCustomerOut{Score: r0, Approved: r1}, nil")
}

func TestWrapInference(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)
	byID := steps(t, "../../../example")

	// an ID beside the attribute spec, inputs inferred from the parameters
	assert.Contains(t, byID, api.StepID("rate-customer-v2"))
	assert.Contains(t, text, "type argyllRateCustomerIn struct {\n\tCustomerId")

	// both sides inferred, from named parameters and named results
	assert.Contains(t, byID, api.StepID("grade-customer"))
	assert.Contains(t, text,
		"type argyllGradeCustomerIn struct {\n\tCustomerId")
	assert.Contains(t, text,
		"type argyllGradeCustomerOut struct {\n\tScore")
}

func TestZeroOutputStep(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, steps(t, "../../../example"), api.StepID("audit"))
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

	attrs := steps(t, "../../../example")["enroll"].Attributes
	assert.Equal(t, api.TypeObject, attrs["address"].Type)
	assert.Equal(t, api.TypeObject, attrs["limits"].Type)
}

func TestFieldTags(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	attrs := steps(t, "../../../example")["enroll"].Attributes
	assert.Equal(t, api.TypeString, attrs["iso_currency"].Type)
	assert.Contains(t, text, `codec.Field("iso_currency"`)

	// a tag of "-" keeps the field off the wire entirely
	assert.NotContains(t, text, "scratch")
	assert.NotContains(t, text, "Scratch")
}

func TestAttributeProps(t *testing.T) {
	attrs := steps(t, "../../../example")["charge-card-v2"].Attributes

	// a for_each attribute is declared as an array, whatever the Go type
	assert.Equal(t, api.TypeArray, attrs["order_id"].Type)
	assert.True(t, attrs["order_id"].Required.ForEach)
	assert.Equal(t, api.InputCollectAll, attrs["order_id"].Required.Collect)

	assert.True(t, attrs["note"].IsOptional())
	assert.Equal(t, int64(5000), attrs["currency"].Optional.Deadline)
	assert.Equal(t, "flow_id", attrs["flow"].Meta.Key)
	assert.Equal(t, "lua", attrs["amount"].Required.Match.Language)

	// a default or const value reaches the engine as JSON
	assert.Equal(t, `"USD"`, attrs["currency"].Optional.Default)
	assert.Equal(t, `"stripe"`, attrs["gateway"].Const.Value)
}

func TestStepProps(t *testing.T) {
	byID := steps(t, "../../../example")
	step := byID["charge-card-v2"]

	assert.Equal(t, api.Name("Charge Card (v2)"), step.Name)
	assert.Equal(t, int64(2500), step.HTTP.Invoke.Timeout)
	assert.Equal(t, "lua", step.Predicate.Language)

	// the endpoints are paths until the step server knows its own host
	assert.Equal(t, "/charge-card-v2", step.HTTP.Invoke.Endpoint)
	assert.Equal(t, "/health", step.HTTP.Health)

	// the conventional ID and name are gone entirely
	assert.NotContains(t, byID, api.StepID("charge-card"))
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
			wants: []string{
				`//argyll:labels takes key:value options, got "domain"`,
			},
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
			wants:   []string{`unknown property "omitempty" on Amount`},
		},
		"tag name": {
			pattern: "./testdata/badtagname",
			wants:   []string{`bad attribute name "order amount"`},
		},
		"step ID": {
			pattern: "./testdata/badid",
			wants:   []string{`bad step ID "Charge_Card"`},
		},
		"step attributes": {
			pattern: "./testdata/badstepattrs",
			wants:   []string{"//argyll:step takes no attribute spec"},
		},
		"step option": {
			pattern: "./testdata/badstepopt",
			wants:   []string{`unknown property "domain"`},
		},
		"field attributes": {
			pattern: "./testdata/badfieldattrs",
			wants:   []string{"Amount is a field, so it names one attribute"},
		},
		"role mismatch": {
			pattern: "./testdata/badroles",
			wants:   []string{`"default" needs role "optional" on Amount`},
		},
		"output role": {
			pattern: "./testdata/badoutopt",
			wants:   []string{`an output takes no role "optional" on Score`},
		},
		"option value": {
			pattern: "./testdata/badoptvalue",
			wants:   []string{`"default" is not key:value`},
		},
		"for each role": {
			pattern: "./testdata/badforeach",
			wants:   []string{`"for_each" needs role "required" on Amount`},
		},
		"unnamed result": {
			pattern: "./testdata/badinfer",
			wants:   []string{"Add result 1 is unnamed"},
		},
		"unnamed parameter": {
			pattern: "./testdata/badinferin",
			wants:   []string{"Add parameter 1 is unnamed"},
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
	byID := steps(t, "../../../example")

	assert.Equal(t, api.Labels{
		"description": "score a customer for risk",
		"domain":      "risk",
	}, byID["calculate-risk"].Labels)

	// a step without labels directives declares none
	assert.Empty(t, byID["greet"].Labels)
}

func render(t *testing.T, pattern string) ([]byte, error) {
	t.Helper()
	pkgs, err := generator.Load(".", pattern)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	return generator.Render(pkgs[0])
}

// steps decodes the specifications the generator emitted, which are the same
// bytes it validated and the same bytes the engine will receive
func steps(t *testing.T, pattern string) map[api.StepID]*api.Step {
	t.Helper()
	src, err := render(t, pattern)
	assert.NoError(t, err)

	res := map[api.StepID]*api.Step{}
	for _, m := range specPattern.FindAllStringSubmatch(string(src), -1) {
		spec, err := strconv.Unquote(m[1])
		assert.NoError(t, err)

		var step api.Step
		assert.NoError(t, json.Unmarshal([]byte(spec), &step))
		assert.NoError(t, step.Validate())
		res[step.ID] = &step
	}
	return res
}
