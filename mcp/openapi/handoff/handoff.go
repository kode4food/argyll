package handoff

import (
	"encoding/json"
	"strings"

	"github.com/kode4food/argyll/mcp/internal/guidance"
)

func Guidance() string {
	text, err := guidance.Read("openapi-ingestion.md")
	if err != nil {
		return ""
	}
	return text
}

func Prompt(payload any) string {
	data, _ := json.Marshal(payload)
	g := Guidance()
	if g == "" {
		return string(data)
	}
	return strings.Replace(g, "{{payload}}", string(data), 1)
}
