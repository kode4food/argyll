package handoff

import (
	"strings"
	"testing"
)

func TestGuidance(t *testing.T) {
	text := Guidance()
	if !strings.Contains(text, "analyze_openapi_contract") {
		t.Fatalf("expected guidance to mention analyze_openapi_contract")
	}
}

func TestPromptInjectsPayload(t *testing.T) {
	prompt := Prompt(map[string]string{"service": "customer-contact"})
	if !strings.Contains(prompt, `"service":"customer-contact"`) {
		t.Fatalf("expected prompt to include serialized payload")
	}
}
