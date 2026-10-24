package script

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// QBESelector returns an Argyll Expressions match script for a query by example
// A Step matches when it carries every tag of any one term
func QBESelector(qbe api.SpaceQuery) *api.ScriptConfig {
	terms := make([]string, 0, len(qbe))
	for _, term := range qbe {
		terms = append(terms, termScript(term, len(qbe) > 1))
	}
	return &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Script:   strings.Join(terms, " ||\n"),
	}
}

// termScript ANDs a term's tags, grouping them in parentheses when the term
// sits beside alternatives
func termScript(term api.SpaceQueryTerm, group bool) string {
	clauses := make([]string, 0, len(term))
	for _, tag := range slices.Sorted(slices.Values(term)) {
		clauses = append(clauses, fmt.Sprintf(
			"%s in value.%s", argexString(tag), MatchTags,
		))
	}
	res := strings.Join(clauses, " &&\n    ")
	if group {
		return "(" + res + ")"
	}
	return res
}

func argexString(value string) string {
	res, _ := json.Marshal(value)
	return string(res)
}
