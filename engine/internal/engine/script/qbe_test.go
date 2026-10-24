package script_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestQBESelector(t *testing.T) {
	cfg := script.QBESelector(api.SpaceQuery{
		{"domain:payments", "tier:gold"},
	})
	assert.Equal(t, &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Script: "\"domain:payments\" in value.tags &&\n    " +
			"\"tier:gold\" in value.tags",
	}, cfg)
}

func TestQBESelectorSingleTag(t *testing.T) {
	cfg := script.QBESelector(api.SpaceQuery{{"example"}})
	assert.Equal(t, &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Script:   `"example" in value.tags`,
	}, cfg)
}

func TestQBESelectorSortsTags(t *testing.T) {
	cfg := script.QBESelector(api.SpaceQuery{{"b", "a"}})
	assert.Equal(t,
		"\"a\" in value.tags &&\n    \"b\" in value.tags", cfg.Script)
}

func TestQBESelectorGroupsAlternatives(t *testing.T) {
	cfg := script.QBESelector(api.SpaceQuery{
		{"domain:risk", "language:lua"},
		{"domain:payments"},
	})
	assert.Equal(t, &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Script: "(\"domain:risk\" in value.tags &&\n    " +
			"\"language:lua\" in value.tags) ||\n" +
			"(\"domain:payments\" in value.tags)",
	}, cfg)
}

func TestQBESelectorMatches(t *testing.T) {
	env := script.NewArgExEnv()
	doc := func(tags api.Tags) map[string]any {
		values := make([]any, len(tags))
		for i, tag := range tags {
			values[i] = tag
		}
		return map[string]any{script.MatchTags: values}
	}
	matches := func(qbe api.SpaceQuery, tags api.Tags) bool {
		comp, err := env.Compile(script.MatchStep, script.QBESelector(qbe))
		assert.NoError(t, err)
		res, err := env.EvaluateMatch(comp, doc(tags))
		assert.NoError(t, err)
		return res
	}

	t.Run("a step carrying the tag", func(t *testing.T) {
		qbe := api.SpaceQuery{{"domain:risk"}}
		assert.True(t, matches(qbe, api.Tags{"domain:risk"}))
		assert.False(t, matches(qbe, api.Tags{"domain:orders"}))
	})

	t.Run("extra tags on the step do not matter", func(t *testing.T) {
		qbe := api.SpaceQuery{{"domain:risk"}}
		assert.True(t, matches(qbe, api.Tags{
			"domain:risk", "domain:payments", "example",
		}))
	})

	t.Run("every tag in a term is required", func(t *testing.T) {
		qbe := api.SpaceQuery{{"domain:risk", "tier:gold"}}
		assert.True(t, matches(qbe, api.Tags{
			"domain:risk", "tier:gold",
		}))
		assert.False(t, matches(qbe, api.Tags{"domain:risk"}))
	})

	t.Run("any one term is enough", func(t *testing.T) {
		qbe := api.SpaceQuery{
			{"domain:risk", "language:lua"},
			{"domain:payments", "language:lua"},
		}
		assert.True(t, matches(qbe, api.Tags{
			"domain:risk", "language:lua",
		}))
		assert.True(t, matches(qbe, api.Tags{
			"domain:payments", "language:lua",
		}))
		assert.False(t, matches(qbe, api.Tags{
			"domain:risk", "domain:payments",
		}))
		assert.False(t, matches(qbe, api.Tags{"language:lua"}))
	})

	t.Run("a step with no tags", func(t *testing.T) {
		qbe := api.SpaceQuery{{"domain:risk"}}
		assert.False(t, matches(qbe, api.Tags{}))
	})
}

func TestQBESelectorEscapes(t *testing.T) {
	zeroWidth := string(rune(0x200b))
	tag := "new\nline" + zeroWidth + `quo"te\slash`
	cfg := script.QBESelector(api.SpaceQuery{{tag}})
	assert.Equal(t,
		`"new\nline`+zeroWidth+`quo\"te\\slash" in value.tags`, cfg.Script)

	env := script.NewArgExEnv()
	comp, err := env.Compile(script.MatchStep, cfg)
	assert.NoError(t, err)
	matched, err := env.EvaluateMatch(comp, map[string]any{
		script.MatchTags: []any{tag},
	})
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestQBESelectorEscapesMarkup(t *testing.T) {
	tag := `a<b>c&d`
	cfg := script.QBESelector(api.SpaceQuery{{tag}})

	env := script.NewArgExEnv()
	comp, err := env.Compile(script.MatchStep, cfg)
	assert.NoError(t, err)
	matched, err := env.EvaluateMatch(comp, map[string]any{
		script.MatchTags: []any{tag},
	})
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestQBESelectorEscapesControlChars(t *testing.T) {
	// A control char with no short escape falls back to \uXXXX
	tag := "bell\a" + "tab\t"
	cfg := script.QBESelector(api.SpaceQuery{{tag}})
	assert.Equal(t,
		`"bell\u0007tab\t" in value.tags`,
		cfg.Script)

	env := script.NewArgExEnv()
	comp, err := env.Compile(script.MatchStep, cfg)
	assert.NoError(t, err)
	matched, err := env.EvaluateMatch(comp, map[string]any{
		script.MatchTags: []any{tag},
	})
	assert.NoError(t, err)
	assert.True(t, matched)
}
