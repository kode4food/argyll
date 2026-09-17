package guidance_test

import (
	"strings"
	"testing"

	"github.com/kode4food/argyll/mcp/internal/guidance"
)

func TestRead(t *testing.T) {
	text, err := guidance.Read("openapi-ingestion.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "required.match") {
		t.Fatalf("expected OpenAPI guidance to mention required.match")
	}
}

func TestRenderTemplate(t *testing.T) {
	code, err := guidance.RenderTemplate("go-step.tmpl", struct {
		StepName         string
		Method           string
		ScriptLanguage   string
		ScriptBody       string
		Inputs           []string
		Outputs          []string
		IsAsync          bool
		IsExternal       bool
		IsScript         bool
		HasNonPostMethod bool
	}{
		StepName: "example-step",
		Method:   "POST",
		Inputs:   []string{"customer_id"},
		Outputs:  []string{"email"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "example-step") {
		t.Fatalf("expected rendered template to include step name")
	}
}
