package guidance_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"

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

func TestEngineAPIGoalsSchema(t *testing.T) {
	raw, err := guidance.Read("engine-api.yaml")
	if !assert.NoError(t, err) {
		return
	}
	doc, err := openapi3.NewLoader().LoadFromData([]byte(raw))
	if !assert.NoError(t, err) {
		return
	}

	for _, name := range []string{
		"CreateFlowRequest", "ExecutionPlanRequest", "ExecutionPlan",
		"FlowConfig",
	} {
		t.Run(name, func(t *testing.T) {
			schema := doc.Components.Schemas[name].Value
			goals := schema.Properties["goals"].Value

			assert.NoError(t, goals.VisitJSON(decode(t,
				`{"steps":["a","b"],"else":{"steps":["c"]}}`,
			)))
			for _, bad := range []string{
				`{}`, `{"steps":[]}`, `{"steps":["a"],"else":{"steps":[]}}`,
				`{"steps":["a"],"else":{}}`, `["a"]`, `[["a"]]`,
			} {
				assert.Error(t, goals.VisitJSON(decode(t, bad)))
			}
		})
	}
}

func decode(t *testing.T, raw string) any {
	t.Helper()
	var res any
	assert.NoError(t, json.Unmarshal([]byte(raw), &res))
	return res
}
