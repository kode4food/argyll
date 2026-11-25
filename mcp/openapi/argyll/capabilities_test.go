package argyll_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/mcp/openapi/argyll"
)

func TestDefaultsDescribePlannerCapabilities(t *testing.T) {
	got := argyll.Defaults()
	raw, err := json.Marshal(api.MappingConfig{
		Selector: &api.ScriptConfig{Language: api.ScriptLangLua, Source: "return value"},
	})
	assert.NoError(t, err)
	var fields map[string]any
	assert.NoError(t, json.Unmarshal(raw, &fields))
	assert.Contains(t, fields, got.AttributeMapping.TransformField)
	if !got.AttributeMapping.Supported {
		t.Fatalf("expected attribute mapping support")
	}
	if got.RequiredMatch.Location != "required.match" {
		t.Fatalf("unexpected required match location: %s", got.RequiredMatch.Location)
	}
	if got.EndpointArgs.PlaceholderSyntax != "{attribute_name}" {
		t.Fatalf("unexpected endpoint placeholder syntax: %s", got.EndpointArgs.PlaceholderSyntax)
	}
}
