package script

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// QBESelector returns a Lua match script equivalent to the query by example,
// ORing the values of each label key and ANDing the keys
func QBESelector(qbe api.SpaceQuery) *api.ScriptConfig {
	clauses := make([]string, 0, len(qbe))
	for _, key := range slices.Sorted(maps.Keys(qbe)) {
		matches := make([]string, len(qbe[key]))
		for i, value := range qbe[key] {
			matches[i] = fmt.Sprintf("value[%s] == %s",
				luaString(key), luaString(value))
		}
		clause := strings.Join(matches, " or ")
		if len(matches) > 1 {
			clause = "(" + clause + ")"
		}
		clauses = append(clauses, clause)
	}
	return &api.ScriptConfig{
		Language: api.ScriptLangLua,
		Script:   "return " + strings.Join(clauses, " and\n    "),
	}
}

// luaString quotes a label for Lua source. strconv.Quote cannot be used here:
// it escapes non-printable runes as \u, which Lua 5.2 rejects
func luaString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range []byte(value) {
		switch {
		case c == '\\' || c == '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, `\%03d`, c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
