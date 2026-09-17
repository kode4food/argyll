package argyll_test

import (
	"testing"

	"github.com/kode4food/argyll/mcp/openapi/argyll"
)

func TestDefaultsDescribePlannerCapabilities(t *testing.T) {
	got := argyll.Defaults()
	if !got.AttributeMapping.Supported {
		t.Fatalf("expected attribute mapping support")
	}
	if got.RequiredMatch.Location != "required.match" {
		t.Fatalf("unexpected required match location: %s",
			got.RequiredMatch.Location)
	}
	if got.EndpointArgs.PlaceholderSyntax != "{attribute_name}" {
		t.Fatalf(
			"unexpected endpoint placeholder syntax: %s",
			got.EndpointArgs.PlaceholderSyntax,
		)
	}
}
