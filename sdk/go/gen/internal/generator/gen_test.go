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

var specPattern = regexp.MustCompile(`(?m)^\s*Spec:\s+(".*"),$`)

func TestNames(t *testing.T) {
	assert.Equal(t, "customer_id", generator.SnakeCase("CustomerID"))
	assert.Equal(t, "http_server", generator.SnakeCase("HTTPServer"))
	assert.Equal(t, "score", generator.SnakeCase("Score"))
	assert.Equal(t, "calculate-risk", generator.KebabCase("CalculateRisk"))
	assert.Equal(t, "Calculate Risk", generator.TitleCase("CalculateRisk"))
	assert.Equal(t, "CustomerId", generator.ExportedName("customer-id"))
	assert.Equal(t, "Score", generator.ExportedName("score"))
	assert.Empty(t, generator.ExportedName("---"))
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

func TestGeneratedFile(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)

	path := filepath.Join("..", "..", "..", "example",
		generator.GeneratedFile)
	built, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, string(built), string(src))
}

func TestGeneratedSurface(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.Contains(t, text, "func ArgyllSteps() []gen.StepDef")
	assert.Contains(t, text, "codecRiskArgs := codec.Struct(")
	assert.Contains(t, text, "codecRefundCardArgs := codec.Struct(")
	assert.NotContains(t, text, "var codecRiskArgs")
	assert.NotContains(t, text, "\nvar codec")
	assert.NotContains(t, text, "\ntype ScoreCustomerIn")
}

func TestGeneratedServer(t *testing.T) {
	src := "package main\n\n//argyll:step\nfunc Run() {}\n"
	out, err := renderFile(t, src, true)
	assert.NoError(t, err)
	text := string(out)

	assert.Contains(t, text, "func main()")
	assert.Contains(t, text, "gen.Serve(context.Background(), steps...)")
	assert.Contains(t, text, `slog.Info("Argyll step invoked",`)
	assert.Contains(t, text, `Handler: logged("run", gen.Sync(`)
	assert.Contains(t, text, `slog.Any("error", err)`)
	assert.NotContains(t, text, "ArgyllSteps")
}

func TestGeneratedSurfaceLogging(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	assert.NotContains(t, text, `"log/slog"`)
	assert.NotContains(t, text, "logged")
}

func TestGeneratedServerPackage(t *testing.T) {
	_, err := renderSourceWithServer(
		t, "//argyll:step\nfunc Run() {}", true,
	)
	assert.ErrorIs(t, err, generator.ErrServerPackage)
}

func TestGeneratedServerMain(t *testing.T) {
	src := "package main\n\nfunc main() {}\n\n" +
		"//argyll:step\nfunc Run() {}\n"
	_, err := renderFile(t, src, true)
	assert.ErrorIs(t, err, generator.ErrMainDeclared)
}

func TestContractInference(t *testing.T) {
	st := steps(t, "../../../example")["calculate-risk"]
	assert.Equal(t, api.Name("Calculate Risk"), st.Name)
	assert.Equal(t, api.StepTypeService, st.Type)
	assert.Equal(t, api.HandlingMemoized, st.Handling)

	attrs := st.Attributes
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

	st := steps(t, "../../../example")["score-customer"]
	assert.Equal(t, api.TypeString, st.Attributes["customer-id"].Type)
	assert.Contains(t, text,
		"r0, r1, err := ScoreCustomer(in.CustomerId, in.Amount)")
	assert.Contains(t, text,
		"return ScoreCustomerOut{Score: r0, Approved: r1}, nil")
}

func TestWrapInference(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)
	byID := steps(t, "../../../example")

	// an ID beside the attribute spec, inputs inferred from the parameters
	assert.Contains(t, byID, api.StepID("rate-customer-v2"))
	assert.Contains(t, text,
		"type RateCustomerIn struct {\n\t\tCustomerId")

	// both sides inferred, from named parameters and named results
	assert.Contains(t, byID, api.StepID("grade-customer"))
	assert.Contains(t, text,
		"type GradeCustomerIn struct {\n\t\tCustomerId")
	assert.Contains(t, text,
		"type GradeCustomerOut struct {\n\t\tScore")
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

func TestMappingName(t *testing.T) {
	src := "type In struct { " +
		"Value string `argyll:\"outer;mapping:inner\"` }\n" +
		"//argyll:step\nfunc Run(in In) {}\n"
	out, err := renderSource(t, src)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `codec.Field("inner"`)
}

func TestAttributeScripts(t *testing.T) {
	src := "type In struct {\n" +
		"Default string `argyll-match:\"$.ready\" " +
		"argyll-mapping:\"value = value; return value\"`\n" +
		"Lua string `argyll-match:\"lua:value = true; return value\"`\n" +
		"}\n" +
		"type Out struct { Value string " +
		"`argyll-mapping:\"jpath:$\"` }\n" +
		"//argyll:step\nfunc Run(in In) Out { return Out{} }\n"
	out, err := renderSource(t, src)
	assert.NoError(t, err)
	text := string(out)
	assert.Contains(t, text, `\"match\":{`+
		`\"language\":\"jpath\",\"script\":\"$.ready\"}`)
	assert.Contains(t, text, `\"script\":{`+
		`\"language\":\"lua\",`+
		`\"script\":\"value = value; return value\"}`)
	assert.Contains(t, text, `\"match\":{`+
		`\"language\":\"lua\",`+
		`\"script\":\"value = true; return value\"}`)
	assert.Contains(t, text, `\"script\":{`+
		`\"language\":\"jpath\",\"script\":\"$\"}`)
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

func TestStepConfig(t *testing.T) {
	byID := steps(t, "../../../example")
	st := byID["charge-card-v2"]

	assert.Equal(t, api.Name("Charge Card (v2)"), st.Name)
	assert.Equal(t, int64(2500), st.HTTP.Invoke.Timeout)
	assert.Equal(t, api.ScriptLangLua, st.Predicate.Language)

	// the endpoints are paths until the step server knows its own host
	assert.Equal(t, "/charge-card-v2", st.HTTP.Invoke.Endpoint)
	assert.Equal(t, "/health", st.HTTP.Health)

	// the conventional ID and name are gone entirely
	assert.NotContains(t, byID, api.StepID("charge-card"))
}

func TestPredicate(t *testing.T) {
	src := "//argyll:step one\n" +
		"//argyll:predicate jpath:$.active\n" +
		"func One() {}\n" +
		"//argyll:step two\n" +
		"//argyll:predicate lua:ready = true; return ready\n" +
		"func Two() {}\n" +
		"//argyll:step three\n" +
		"//argyll:predicate custom:ready()\n" +
		"func Three() {}\n"
	out, err := renderSource(t, src)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `\"predicate\":{`+
		`\"language\":\"jpath\",\"script\":\"$.active\"}`)
	assert.Contains(t, string(out), `\"predicate\":{`+
		`\"language\":\"lua\",`+
		`\"script\":\"ready = true; return ready\"}`)
	assert.Contains(t, string(out), `\"predicate\":{`+
		`\"language\":\"lua\",\"script\":\"custom:ready()\"}`)
}

func TestWorkConfig(t *testing.T) {
	src := "//argyll:step\n" +
		"//argyll:work backoff_type:exponential;max_retries:3\n" +
		"//argyll:work init_backoff:100;max_backoff:5000;parallelism:4\n" +
		"func Run() {}\n"
	out, err := renderSource(t, src)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `\"work_config\":{`+
		`\"backoff_type\":\"exponential\",\"max_retries\":3,`+
		`\"init_backoff\":100,\"max_backoff\":5000,\"parallelism\":4}`)
}

func TestRecursiveCodec(t *testing.T) {
	src, err := render(t, "../../../example")
	assert.NoError(t, err)
	text := string(src)

	// a self-referential codec cannot be a plain var initializer
	assert.Contains(t, text, "var codecNodeImpl codec.Codec[Node]")
	assert.Contains(t, text,
		"codecNode := codec.Ref(&codecNodeImpl)")
	assert.Contains(t, text, "codecNodeImpl = codec.Struct(")
	assert.Contains(t, text, "codec.Slice(codecNode)")
	assert.Contains(t, text, "codec.Optional(codecNode)")

	// non-recursive codecs stay plain
	assert.Contains(t, text, "codecRiskArgs := codec.Struct(")
}

func TestNoDirectives(t *testing.T) {
	src, err := render(t, "./testdata/nodirectives")
	assert.NoError(t, err)
	assert.Nil(t, src)

	src, err = renderSource(t, "//argyll:stepper\nfunc Run() {}")
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
			wants:   []string{"TooManyArgs", "zero or one argument struct"},
		},
		"wrap arity": {
			pattern: "./testdata/badwrap",
			wants:   []string{"Add", "declares 2 outputs but returns 1"},
		},
		"tag syntax": {
			pattern: "./testdata/badsteptags",
			wants: []string{
				"//argyll:tags has an empty tag",
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

func TestAdditionalDiagnostics(t *testing.T) {
	tests := map[string]struct {
		src  string
		want string
	}{
		"step input is a struct": {
			src:  "//argyll:step\nfunc Bad(in int) {}",
			want: "int is not a struct",
		},
		"step has at most one input": {
			src: "//argyll:step\n" +
				"func Bad(left, right struct{}) {}",
			want: "takes zero or one argument struct",
		},
		"step output is a struct": {
			src: "//argyll:step\n" +
				"func Bad(in struct{}) int { return 0 }",
			want: "int is not a struct",
		},
		"step has at most two results": {
			src: "//argyll:step\n" +
				"func Bad(in struct{}) (int, int) { return 0, 0 }",
			want: "returns more than an outputs struct and an error",
		},
		"directive is not allowed on a method": {
			src: "type T struct{}\n" +
				"//argyll:step\nfunc (T) Bad(in struct{}) {}",
			want: "must be a plain generic-free function",
		},
		"directive is not allowed on a generic function": {
			src: "//argyll:step\n" +
				"func Bad[T any](in struct{}) {}",
			want: "must be a plain generic-free function",
		},
		"compensator exists": {
			src: "//argyll:step\n//argyll:compensate Missing\n" +
				"func Run() {}",
			want: "compensator Missing not found",
		},
		"compensator takes one struct": {
			src: "type In struct{}\ntype Out struct{}\n" +
				"//argyll:step\n//argyll:compensate Undo\n" +
				"func Run(In) Out { return Out{} }\n" +
				"func Undo(Out, In) {}",
			want: "compensator Undo takes zero or one argument struct",
		},
		"embedded structs do not collide on a field": {
			src: "type In struct{ Status string }\n" +
				"type Out struct{ Status string `argyll:\"state\"` }\n" +
				"type Args struct{ In; Out }\n" +
				"//argyll:step\nfunc Run(in Args) {}",
			want: "ambiguous embedded field: field Status",
		},
		"embedded structs do not collide on an attribute": {
			src: "type In struct{ Status string }\n" +
				"type Out struct{ Other string `argyll:\"status\"` }\n" +
				"type Args struct{ In; Out }\n" +
				"//argyll:step\nfunc Run(in Args) {}",
			want: "ambiguous embedded field: attribute \"status\"",
		},
		"compensator fields are selected": {
			src: "type In struct { Value string }\n" +
				"type UndoArgs struct { Value string }\n" +
				"//argyll:step\n//argyll:compensate Undo\n" +
				"func Run(In) {}\nfunc Undo(UndoArgs) {}",
			want: "field value is not compensated",
		},
		"compensator field types match": {
			src: "type In struct { Value string " +
				"`argyll:\"compensated:true\"` }\n" +
				"type UndoArgs struct { Value int }\n" +
				"//argyll:step\n//argyll:compensate Undo\n" +
				"func Run(In) {}\nfunc Undo(UndoArgs) {}",
			want: "field value has type int; want string",
		},
		"compensator returns only error": {
			src: "//argyll:step\n//argyll:compensate Undo\n" +
				"func Run() {}\nfunc Undo() int { return 0 }",
			want: "must return nothing or error",
		},
		"compensator directive names a function": {
			src:  "//argyll:step\n//argyll:compensate\nfunc Run() {}",
			want: "needs a function name",
		},
		"compensator timeout is milliseconds": {
			src: "//argyll:step\n//argyll:compensate Undo;timeout:soon\n" +
				"func Run() {}\nfunc Undo() {}",
			want: "timeout\" needs milliseconds",
		},
		"compensator properties are known": {
			src: "//argyll:step\n//argyll:compensate Undo;retries:3\n" +
				"func Run() {}\nfunc Undo() {}",
			want: `unknown property "retries"`,
		},
		"wrap input count matches": {
			src: "//argyll:wrap (left) -> ()\n" +
				"func Bad(left, right int) {}",
			want: "declares 1 inputs but takes 2",
		},
		"wrap inputs are parenthesized": {
			src:  "//argyll:wrap () -> result\nfunc Bad() int { return 0 }",
			want: "attributes \"result\" are not parenthesized",
		},
		"wrap inputs have a closing parenthesis": {
			src:  "//argyll:wrap (left -> ()\nfunc Bad(left int) {}",
			want: "attributes \"(left\" are not parenthesized",
		},
		"wrap inputs have names": {
			src:  "//argyll:wrap (left, ) -> ()\nfunc Bad(left int) {}",
			want: "bad attribute name",
		},
		"handling is not a generator property": {
			src: "//argyll:step;handling:memoized\n" +
				"func Bad(in struct{}) {}",
			want: `unknown property "handling"`,
		},
		"memoize takes no value": {
			src: "//argyll:step\n//argyll:memoize sometimes\n" +
				"func Bad() {}",
			want: "memoize takes no value",
		},
		"memoize and compensate are exclusive": {
			src: "//argyll:step\n//argyll:memoize\n" +
				"//argyll:compensate Undo\nfunc Bad() {}\nfunc Undo() {}",
			want: "memoize and //argyll:compensate are mutually exclusive",
		},
		"timeout is milliseconds": {
			src: "//argyll:step\n//argyll:http timeout:soon\n" +
				"func Bad(in struct{}) {}",
			want: "timeout\" needs milliseconds",
		},
		"http properties need http": {
			src:  "//argyll:step;timeout:2\nfunc Bad() {}",
			want: `unknown property "timeout"`,
		},
		"parallelism is an integer": {
			src: "//argyll:step\n//argyll:work parallelism:many\n" +
				"func Bad(in struct{}) {}",
			want: "parallelism\" needs an integer",
		},
		"work needs options": {
			src:  "//argyll:step\n//argyll:work\nfunc Bad() {}",
			want: "takes key:value options",
		},
		"work properties need work": {
			src: "//argyll:step\n//argyll:http parallelism:2\n" +
				"func Bad() {}",
			want: `unknown property "parallelism"`,
		},
		"predicate needs a script": {
			src:  "//argyll:step\n//argyll:predicate\nfunc Bad() {}",
			want: "predicate needs a script",
		},
		"predicate does not repeat": {
			src: "//argyll:step\n//argyll:predicate return true\n" +
				"//argyll:predicate return false\nfunc Bad() {}",
			want: "predicate repeats",
		},
		"predicate property needs directive": {
			src: "//argyll:step\n" +
				"//argyll:http predicate:return true\nfunc Bad() {}",
			want: `unknown property "predicate"`,
		},
		"http needs options": {
			src: "//argyll:step\n//argyll:http\n" +
				"func Bad(in struct{}) {}",
			want: "takes key:value options",
		},
		"http has no head": {
			src: "//argyll:step\n//argyll:http stray;timeout:2\n" +
				"func Bad(in struct{}) {}",
			want: "takes key:value options",
		},
		"props are not supported": {
			src: "//argyll:step\n//argyll:props timeout:2\n" +
				"func Bad(in struct{}) {}",
			want: "//argyll:props is not supported",
		},
		"roles are known": {
			src: "type In struct { Value string `argyll:\";role:weird\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "unknown role",
		},
		"collect needs an input": {
			src: "type In struct { Value string " +
				"`argyll:\";role:const;collect:all\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "collect\" needs role",
		},
		"for each must be true": {
			src: "type In struct { Value string " +
				"`argyll:\";for_each:sometimes\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "for_each\" needs true or false",
		},
		"match needs a required input": {
			src: "type In struct { Value *string " +
				"`argyll-match:\"return true\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "argyll-match\" needs role",
		},
		"mapping needs an input or output": {
			src: "type In struct { Value string " +
				"`argyll:\";role:const;mapping:x\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "mapping\" needs role",
		},
		"wrap input type is supported": {
			src:  "//argyll:wrap (value) -> ()\nfunc Bad(value chan int) {}",
			want: "unsupported attribute type",
		},
		"wrap output type is supported": {
			src: "//argyll:wrap () -> (value)\n" +
				"func Bad() chan int { return nil }",
			want: "unsupported attribute type",
		},
		"nested types are supported": {
			src: "type In struct { Value []chan int }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "unsupported attribute type",
		},
		"basic types are supported": {
			src: "type In struct { Value complex64 }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "unsupported attribute type",
		},
		"step directive options are valid": {
			src:  "//argyll:step;bad\nfunc Bad(in struct{}) {}",
			want: "invalid argyll option",
		},
		"http options are valid": {
			src: "//argyll:step\n//argyll:http;bad\n" +
				"func Bad(in struct{}) {}",
			want: "invalid argyll option",
		},
		"default needs an optional input": {
			src: "type In struct { Value string `argyll:\";default:x\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "default\" needs role",
		},
		"value needs a const": {
			src: "type In struct { Value string `argyll:\";value:x\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "value\" needs role",
		},
		"key needs metadata": {
			src: "type In struct { Value string `argyll:\";key:x\"` }\n" +
				"//argyll:step\nfunc Bad(in In) {}",
			want: "key\" needs role",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := renderSource(t, tt.src)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestStepShapesAndProps(t *testing.T) {
	src := "import \"errors\"\n" +
		"type In struct {\n" +
		"Required string `argyll:\";mapping:source\"`\n" +
		"Optional *string " +
		"`argyll:\";collect:all;for_each:true;mapping:fallback\"`\n" +
		"}\n" +
		"type Out struct { Result string `argyll:\";mapping:result\"` }\n" +
		"//argyll:step\nfunc Empty() {}\n" +
		"//argyll:step\nfunc EmptyError() error { " +
		"return errors.New(\"bad\") }\n" +
		"//argyll:step\nfunc Source() Out { return Out{} }\n" +
		"//argyll:step\nfunc SourceError() (Out, error) { " +
		"return Out{}, nil }\n" +
		"//argyll:step\nfunc NoResult(in In) {}\n" +
		"//argyll:step\nfunc ErrorOnly(in In) error { " +
		"return errors.New(\"bad\") }\n" +
		"//argyll:step\nfunc ResultOnly(in In) Out { return Out{} }\n" +
		"//argyll:step\nfunc ResultError(in In) (Out, error) { " +
		"return Out{}, nil }\n"

	out, err := renderSource(t, src)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "Empty()")
	assert.Contains(t, string(out), "return struct{}{}, EmptyError()")
	assert.Contains(t, string(out), "return Source(), nil")
	assert.Contains(t, string(out), "return SourceError()")
	assert.Contains(t, string(out), "NoResult(in)")
	assert.Contains(t, string(out), "return struct{}{}, ErrorOnly(in)")
	assert.Contains(t, string(out), "return ResultOnly(in), nil")
	assert.Contains(t, string(out), "return ResultError(in)")
}

func TestCompensate(t *testing.T) {
	src := "type In struct { Value string `argyll:\"compensated:true\"` }\n" +
		"type Out struct { ID string `argyll:\"compensated:true\"` }\n" +
		"type UndoArgs struct { ID string }\n" +
		"//argyll:step\n//argyll:compensate Undo;timeout:2500\n" +
		"func Run(in In) (Out, error) { return Out{}, nil }\n" +
		"func Undo(UndoArgs) error { return nil }\n" +
		"//argyll:wrap\n//argyll:compensate Unwrap\n" +
		"func Wrapped(value string) (result int) { return 0 }\n" +
		"func Unwrap(result int) {}\n"
	src += "//argyll:step\n//argyll:compensate Reset\n" +
		"func Empty() {}\nfunc Reset() {}\n"

	out, err := renderSource(t, src)
	assert.NoError(t, err)
	text := string(out)
	assert.Contains(t, text, "/run/compensate")
	assert.Contains(t, text, `\"compensate\":{`+
		`\"endpoint\":\"/run/compensate\",\"timeout\":2500}`)
	assert.Contains(t, text, `\"handling\":\"compensated\"`)
	assert.Contains(t, text, `\"compensated\":true`)
	assert.Contains(t, text, `\"result\":{\"output\":{},\"role\":`+
		`\"output\",\"type\":\"number\",\"compensated\":true}`)
	assert.Contains(t, text, "type WrappedCompIn struct")
	assert.NotContains(t, text, "CompensateIn")
	assert.Contains(t, text, "return Undo(in)")
	assert.Contains(t, text, "Unwrap(in.Result)\n")
	assert.Contains(t, text, "Reset()\n")
	assert.Contains(t, text, "Compensate: gen.Compensate(")
}

func TestEmbeddedFields(t *testing.T) {
	src := "type In struct { Value string `argyll:\"compensated:true\"` }\n" +
		"type Out struct { ID string `argyll:\"compensated:true\"` }\n" +
		"type UndoArgs struct { In; Out }\n" +
		"//argyll:step\n//argyll:compensate Undo\n" +
		"func Run(in In) (Out, error) { return Out{}, nil }\n" +
		"func Undo(UndoArgs) error { return nil }\n"

	out, err := renderSource(t, src)
	assert.NoError(t, err)
	text := string(out)
	assert.Contains(t, text, `codec.Field("value", codec.Text[string](),`)
	assert.Contains(t, text, `codec.Field("id", codec.Text[string](),`)
	assert.Contains(t, text, "return &v.Value")
	assert.Contains(t, text, "return &v.ID")
	assert.NotContains(t, text, `codec.Field("in"`)
	assert.Contains(t, text, "return Undo(in)")
}

func TestMemoization(t *testing.T) {
	out, err := renderSource(
		t, "//argyll:step\n//argyll:memoize\nfunc Run() {}",
	)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `\"handling\":\"memoized\"`)
}

func TestGenerateIsIdempotent(t *testing.T) {
	_, err := generator.Generate(".", false, "../../../example")
	assert.NoError(t, err)

	written, err := generator.Generate(".", false, "../../../example")
	assert.NoError(t, err)
	assert.Empty(t, written)
}

func TestGenerateStale(t *testing.T) {
	dir := filepath.Join("testdata", "nodirectives")
	path := filepath.Join(dir, generator.GeneratedFile)
	assert.NoError(t, os.WriteFile(path, []byte("package nodirectives\n"),
		generator.FileMode))

	_, err := generator.Generate(".", false, "./"+dir)
	assert.NoError(t, err)
	assert.NoFileExists(t, path)
}

func TestGenerateTemporary(t *testing.T) {
	path := writeSource(t, "//argyll:step\nfunc Run(in struct{}) {}")
	written, err := generator.Generate(".", false, "file="+path)
	assert.NoError(t, err)
	assert.Len(t, written, 1)

	written, err = generator.Generate(".", false, "file="+path)
	assert.NoError(t, err)
	assert.Empty(t, written)
}

func TestGenerateError(t *testing.T) {
	path := writeSource(t, "//argyll:step\nfunc Bad(in int) {}")
	_, err := generator.Generate(".", false, "file="+path)
	assert.Error(t, err)
}

func TestLoadError(t *testing.T) {
	_, err := generator.Load(filepath.Join(t.TempDir(), "missing"), ".")
	assert.ErrorIs(t, err, generator.ErrLoadFailed)
}

func TestTags(t *testing.T) {
	byID := steps(t, "../../../example")

	assert.Equal(t, api.Tags{"domain:risk", "scoring"},
		byID["calculate-risk"].Tags)

	// a step without tags directives declares none
	assert.Empty(t, byID["greet"].Tags)
}

func TestDescription(t *testing.T) {
	byID := steps(t, "../../../example")

	assert.Equal(t, "score a customer for risk",
		byID["calculate-risk"].Description)
	assert.Empty(t, byID["greet"].Description)
}

func TestDescriptionRepeats(t *testing.T) {
	src := "//argyll:step\n" +
		"//argyll:description one\n" +
		"//argyll:description two\n" +
		"func Run() {}"
	_, err := renderSource(t, src)
	assert.ErrorContains(t, err, "//argyll:description repeats")
}

func TestDescriptionEmpty(t *testing.T) {
	src := "//argyll:step\n//argyll:description\nfunc Run() {}"
	_, err := renderSource(t, src)
	assert.ErrorContains(t, err, "//argyll:description needs a description")
}

func TestTagsRepeatedAndSorted(t *testing.T) {
	src := "//argyll:step\n" +
		"//argyll:tags domain:risk; example\n" +
		"//argyll:tags domain:payments; example\n" +
		"func Run() {}"
	out, err := renderSource(t, src)
	assert.NoError(t, err)
	assert.Contains(t, string(out),
		`\"tags\":[\"domain:payments\",\"domain:risk\",\"example\"]`)
}

func render(t *testing.T, pattern string) ([]byte, error) {
	t.Helper()
	pkgs, err := generator.Load(".", pattern)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	return generator.Render(pkgs[0], false)
}

func renderSource(t *testing.T, src string) ([]byte, error) {
	return renderSourceWithServer(t, src, false)
}

func renderSourceWithServer(
	t *testing.T, src string, server bool,
) ([]byte, error) {
	t.Helper()
	path := writeSource(t, src)
	pkgs, err := generator.Load(".", "file="+path)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	return generator.Render(pkgs[0], server)
}

func renderFile(t *testing.T, src string, server bool) ([]byte, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "steps.go")
	assert.NoError(t, os.WriteFile(path, []byte(src), generator.FileMode))
	pkgs, err := generator.Load(".", "file="+path)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	return generator.Render(pkgs[0], server)
}

func writeSource(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "steps.go")
	content := "package sample\n\n" + src + "\n"
	assert.NoError(t, os.WriteFile(path, []byte(content), generator.FileMode))
	return path
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

		var st api.Step
		assert.NoError(t, json.Unmarshal([]byte(spec), &st))
		assert.NoError(t, st.Validate())
		res[st.ID] = &st
	}
	return res
}
