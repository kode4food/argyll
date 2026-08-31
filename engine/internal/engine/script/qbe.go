package script

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// QBESelector returns a JPath match script equivalent to the query by example,
// ANDing its tags so a step matches when it carries every one of them
func QBESelector(qbe api.SpaceQuery) *api.ScriptConfig {
	tags := jpathString(MatchTags)
	clauses := make([]string, 0, len(qbe))
	for _, tag := range slices.Sorted(slices.Values(qbe)) {
		clauses = append(clauses,
			fmt.Sprintf("$[%s][%s]", tags, jpathString(tag)))
	}
	return &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   strings.Join(clauses, " &&\n    "),
	}
}

// jpathString quotes a tag for JPath source, whose string literals take JSON
// escapes. Marshalling a string cannot fail
func jpathString(value string) string {
	res, _ := json.Marshal(value)
	return string(res)
}
