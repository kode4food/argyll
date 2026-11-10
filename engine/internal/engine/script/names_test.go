package script_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestDerivedNames(t *testing.T) {
	envs := map[string]script.Environment{
		api.ScriptLangLua:    script.NewLuaEnv(),
		api.ScriptLangArgyll: script.NewArgExEnv(),
	}

	tests := []struct {
		name     string
		inputs   api.Args
		scripts  map[string]string
		expected any
	}{
		{
			name:   "tail_binds_bare",
			inputs: api.Args{"urn:acme:billing/order": 7},
			scripts: map[string]string{
				api.ScriptLangLua:    `return {out = order}`,
				api.ScriptLangArgyll: `{out: order}`,
			},
			expected: 7,
		},
		{
			name: "clean_name_wins",
			inputs: api.Args{
				"order":          1,
				"urn:acme:order": 2,
			},
			scripts: map[string]string{
				api.ScriptLangLua:    `return {out = order}`,
				api.ScriptLangArgyll: `{out: order}`,
			},
			expected: 1,
		},
		{
			name: "shared_tail_binds_none",
			inputs: api.Args{
				"urn:acme:billing:order":  1,
				"urn:acme:shipping:order": 2,
			},
			scripts: map[string]string{
				api.ScriptLangLua:    `return {out = order == nil}`,
				api.ScriptLangArgyll: `{out: (order ?? null) == null}`,
			},
			expected: true,
		},
		{
			name: "table_holds_full_and_short_names",
			inputs: api.Args{
				"user_id":        1,
				"urn:acme:order": 10,
			},
			scripts: map[string]string{
				api.ScriptLangLua: `return {out = this["user_id"] == 1 and
					this["urn:acme:order"] == 10 and this["order"] == 10}`,
				api.ScriptLangArgyll: `{out: this["user_id"] == 1 &&
					this["urn:acme:order"] == 10 && this["order"] == 10}`,
			},
			expected: true,
		},
		{
			name:   "digit_tail_binds_none",
			inputs: api.Args{"urn:acme:123": 1},
			scripts: map[string]string{
				api.ScriptLangLua:    `return {out = this["urn:acme:123"]}`,
				api.ScriptLangArgyll: `{out: this["urn:acme:123"]}`,
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		for lang, src := range tt.scripts {
			t.Run(tt.name+"/"+lang, func(t *testing.T) {
				assert.Equal(t, tt.expected, evalInputs(t, evalInputsArgs{
					env:      envs[lang],
					language: lang,
					script:   src,
					inputs:   tt.inputs,
				}))
			})
		}
	}
}
