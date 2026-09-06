package script_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// benchCase pairs a Lua script with the JPath expression meant to decide the
// same thing about the same document
type benchCase struct {
	name  string
	lua   string
	jpath string
}

// compilePadding varies a source so repeated compiles miss the script cache
const compilePadding = 64

// matchDoc is the shape a Space selector sees: stable Step metadata
var matchDoc = map[string]any{
	"tags": []any{
		"domain:payments", "capability:create", "environment:production",
		"market:europe", "tier:gold", "example",
	},
	"type":     "service",
	"handling": "standard",
	"attributes": map[string]any{
		"score": map[string]any{
			"role": "output", "type": "number", "compensated": false,
		},
		"amount": map[string]any{
			"role": "required", "type": "number", "compensated": false,
		},
	},
}

// predicateArgs is the shape a Step predicate sees: its input arguments
var predicateArgs = api.Args{
	"user_info": map[string]any{
		"id": "u-1", "name": "Alice", "account_type": "business",
		"credit_limit": 5000.0,
	},
	"product_info": map[string]any{
		"name": "Professional Laptop", "price": 2499.0, "in_stock": true,
	},
	"quantity": 3.0,
}

// predicateStep names the args so the Lua wrapper binds them as locals
var predicateStep = &api.Step{
	Attributes: api.AttributeSpecs{
		"user_info":    {Role: api.RoleRequired, Type: api.TypeObject},
		"product_info": {Role: api.RoleRequired, Type: api.TypeObject},
		"quantity":     {Role: api.RoleRequired, Type: api.TypeNumber},
	},
}

// matchCases evaluate one document, as Space selectors and attribute
// matches do
var matchCases = []benchCase{
	{
		name:  "SingleTag",
		lua:   `return has(value.tags, "domain:payments")`,
		jpath: `$.tags[?@=="domain:payments"]`,
	},
	{
		name: "TagAnd",
		lua: `return has(value.tags, "domain:payments") and ` +
			`has(value.tags, "environment:production")`,
		jpath: `$.tags[?@=="domain:payments"] && ` +
			`$.tags[?@=="environment:production"]`,
	},
	{
		name: "TagOr",
		lua: `return has(value.tags, "market:europe") or ` +
			`has(value.tags, "market:all")`,
		jpath: `$.tags[?@=="market:europe" || @=="market:all"]`,
	},
	{
		name: "QBETwoTerms",
		lua: `return (has(value.tags, "domain:payments") and ` +
			`has(value.tags, "tier:gold")) or ` +
			`(has(value.tags, "domain:risk") and ` +
			`has(value.tags, "environment:production"))`,
		jpath: `($.tags[?@=="domain:payments"] && ` +
			`$.tags[?@=="tier:gold"]) || ` +
			`($.tags[?@=="domain:risk"] && ` +
			`$.tags[?@=="environment:production"])`,
	},
	{
		name: "TagMiss",
		// worst case for both: full scan of the tag list, no match
		lua:   `return has(value.tags, "domain:nonexistent")`,
		jpath: `$.tags[?@=="domain:nonexistent"]`,
	},
	{
		name: "NestedMetadata",
		lua: `local score = value.attributes.score
return has(value.tags, "domain:payments") and score and
    value.type == "service" and value.handling == "standard" and
    score.role == "output" and score.type == "number" and
    score.compensated == false`,
		jpath: `$.tags[?@=="domain:payments"] && $.type=="service" && ` +
			`$.handling=="standard" && ` +
			`$.attributes.score.role=="output" && ` +
			`$.attributes.score.type=="number" && ` +
			`$.attributes.score.compensated==false`,
	},
	{
		name: "Descendant",
		lua: `for _, a in pairs(value.attributes) do
  for _, v in pairs(a) do
    if v == "number" then return true end
  end
end
return false`,
		jpath: `$.attributes..[?@=="number"]`,
	},
}

// predicateCases evaluate a Step's input arguments
var predicateCases = []benchCase{
	{
		name:  "FieldEquals",
		lua:   `return product_info.name == "Professional Laptop"`,
		jpath: `$.product_info.name=="Professional Laptop"`,
	},
	{
		name: "NumericCompare",
		lua: `return quantity > 1 and ` +
			`user_info.credit_limit >= product_info.price`,
		jpath: `$.quantity>1 && ` +
			`$.user_info.credit_limit>=$.product_info.price`,
	},
	{
		name: "MultiField",
		lua: `return user_info.account_type == "business" and ` +
			`product_info.in_stock and quantity <= 10`,
		jpath: `$.user_info.account_type=="business" && ` +
			`$.product_info.in_stock==true && $.quantity<=10`,
	},
}

// TestBenchCasesAgree keeps the two dialects of each benchmark case
// semantically equivalent, so the timings compare like for like
func TestBenchCasesAgree(t *testing.T) {
	lua := script.NewLuaEnv()
	jp := script.NewJPathEnv()

	for _, c := range matchCases {
		t.Run(c.name, func(t *testing.T) {
			luaC := compileFor(t, lua, script.MatchStep, c.lua)
			jpC := compileFor(t, jp, script.MatchStep, c.jpath)
			luaRes, err := lua.EvaluateMatch(luaC, matchDoc)
			assert.NoError(t, err)
			jpRes, err := jp.EvaluateMatch(jpC, matchDoc)
			assert.NoError(t, err)
			assert.Equal(t, luaRes, jpRes)
		})
	}

	for _, c := range predicateCases {
		t.Run(c.name, func(t *testing.T) {
			luaC := compileFor(t, lua, predicateStep, c.lua)
			jpC := compileFor(t, jp, predicateStep, c.jpath)
			luaRes, err := lua.EvaluatePredicate(luaC,
				predicateStep, predicateArgs)
			assert.NoError(t, err)
			jpRes, err := jp.EvaluatePredicate(jpC,
				predicateStep, predicateArgs)
			assert.NoError(t, err)
			assert.Equal(t, luaRes, jpRes)
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	lua := script.NewLuaEnv()
	jp := script.NewJPathEnv()

	for _, c := range matchCases {
		luaC := compileForBench(b, lua, script.MatchStep, c.lua)
		jpC := compileForBench(b, jp, script.MatchStep, c.jpath)

		b.Run(c.name+"/lua", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := lua.EvaluateMatch(luaC, matchDoc); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(c.name+"/jpath", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := jp.EvaluateMatch(jpC, matchDoc); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPredicate(b *testing.B) {
	lua := script.NewLuaEnv()
	jp := script.NewJPathEnv()

	for _, c := range predicateCases {
		luaC := compileForBench(b, lua, predicateStep, c.lua)
		jpC := compileForBench(b, jp, predicateStep, c.jpath)

		b.Run(c.name+"/lua", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := lua.EvaluatePredicate(luaC, predicateStep,
					predicateArgs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(c.name+"/jpath", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := jp.EvaluatePredicate(jpC, predicateStep,
					predicateArgs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMatchParallel exercises the environments under concurrency, where
// the Lua state pool becomes contended
func BenchmarkMatchParallel(b *testing.B) {
	lua := script.NewLuaEnv()
	jp := script.NewJPathEnv()
	c := matchCases[0]
	luaC := compileForBench(b, lua, script.MatchStep, c.lua)
	jpC := compileForBench(b, jp, script.MatchStep, c.jpath)

	b.Run("lua", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := lua.EvaluateMatch(luaC, matchDoc); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})
	b.Run("jpath", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := jp.EvaluateMatch(jpC, matchDoc); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})
}

// BenchmarkCompile measures per-script compilation, hoisting environment
// construction out of the loop and defeating the script cache with a
// distinct but semantically identical source per iteration
func BenchmarkCompile(b *testing.B) {
	lua := script.NewLuaEnv()
	jp := script.NewJPathEnv()

	for _, c := range matchCases {
		b.Run(c.name+"/lua", func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				i++
				compileForBench(b, lua, script.MatchStep,
					fmt.Sprintf("%s\n-- %d", c.lua, i))
			}
		})
		b.Run(c.name+"/jpath", func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				i++
				compileForBench(b, jp, script.MatchStep, padJPath(c.jpath, i))
			}
		})
	}
}

// BenchmarkNewEnv measures one-time environment construction
func BenchmarkNewEnv(b *testing.B) {
	b.Run("lua", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			script.NewLuaEnv()
		}
	})
	b.Run("jpath", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			script.NewJPathEnv()
		}
	})
}

func compileFor(
	t *testing.T, env script.Environment, st *api.Step, src string,
) script.Compiled {
	t.Helper()
	c, err := env.Compile(st, &api.ScriptConfig{Script: src})
	assert.NoError(t, err)
	return c
}

func compileForBench(
	b *testing.B, env script.Environment, st *api.Step, src string,
) script.Compiled {
	b.Helper()
	c, err := env.Compile(st, &api.ScriptConfig{Script: src})
	if err != nil {
		b.Fatal(err)
	}
	return c
}

// JPath rejects surrounding whitespace, so the padding goes after the first
// root selector, where it changes the text but not the meaning
func padJPath(src string, i int) string {
	pad := strings.Repeat(" ", i%compilePadding)
	return strings.Replace(src, "$", "$"+pad, 1)
}
