package script

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// QBESelector returns a JPath match script equivalent to the query by example,
// ANDing the tags within a term and ORing the terms, so a step matches when it
// carries every tag of any one of them
func QBESelector(qbe api.SpaceQuery) *api.ScriptConfig {
	terms := make([]string, 0, len(qbe))
	for _, term := range qbe {
		terms = append(terms, termScript(term, len(qbe) > 1))
	}
	return &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   strings.Join(terms, " ||\n"),
	}
}

// termScript ANDs a term's tags, grouping them in parentheses when the term
// sits beside alternatives
func termScript(term api.SpaceQueryTerm, group bool) string {
	clauses := make([]string, 0, len(term))
	for _, tag := range slices.Sorted(slices.Values(term)) {
		clauses = append(clauses,
			fmt.Sprintf("$.%s[?@==%s]", MatchTags, jpathString(tag)))
	}
	res := strings.Join(clauses, " &&\n    ")
	if group {
		return "(" + res + ")"
	}
	return res
}

// jpathString quotes a tag for JPath source, whose string literals take JSON
// escapes. Marshalling a string cannot fail
func jpathString(value string) string {
	res, _ := json.Marshal(value)
	return string(res)
}
