package handoff

import (
	"encoding/json"
	"strings"

	guide "github.com/kode4food/argyll/mcp/internal/guidance"
)

func Guidance() string {
	text, err := guide.Read("openapi-ingestion.md")
	if err != nil {
		return ""
	}
	return text
}

func Prompt(payload any) string {
	data, _ := json.Marshal(payload)
	guidance := Guidance()
	if guidance == "" {
		return string(data)
	}
	return strings.Replace(guidance, "{{payload}}", string(data), 1)
}
