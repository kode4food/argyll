package handoff_test

import (
	"strings"
	"testing"

	"github.com/kode4food/argyll/mcp/openapi/handoff"
)

func TestGuidance(t *testing.T) {
	text := handoff.Guidance()
	if !strings.Contains(text, "analyze_openapi_contract") {
		t.Fatalf("expected guidance to mention analyze_openapi_contract")
	}
}

func TestPromptInjectsPayload(t *testing.T) {
	prompt := handoff.Prompt(map[string]string{"service": "customer-contact"})
	if !strings.Contains(prompt, `"service":"customer-contact"`) {
		t.Fatalf("expected prompt to include serialized payload")
	}
}
