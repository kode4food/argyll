package script_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestQBESelector(t *testing.T) {
	cfg := script.QBESelector(api.SpaceQuery{
		"domain": {"payments", "risk"},
		"tier":   {"gold"},
	})
	assert.Equal(t, &api.ScriptConfig{
		Language: api.ScriptLangLua,
		Script: "return (value[\"domain\"] == \"payments\" or " +
			"value[\"domain\"] == \"risk\") and\n    " +
			"value[\"tier\"] == \"gold\"",
	}, cfg)
}

func TestQBESelectorEscapes(t *testing.T) {
	// The zero width space is why strconv.Quote cannot be used here: it
	// escapes non-printable runes in a form Lua 5.2 rejects
	zeroWidth := string(rune(0x200b))
	label := "new\nline" + zeroWidth
	cfg := script.QBESelector(api.SpaceQuery{`quo"te`: {label}})
	assert.Equal(t,
		"return value[\"quo\\\"te\"] == \"new\\010line"+zeroWidth+"\"",
		cfg.Script)

	env := script.NewLuaEnv()
	comp, err := env.Compile(script.MatchStep, cfg)
	assert.NoError(t, err)
	matched, err := env.EvaluateMatch(comp, map[string]any{`quo"te`: label})
	assert.NoError(t, err)
	assert.True(t, matched)
}
