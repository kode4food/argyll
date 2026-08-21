package gen

import (
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// StepDef is the logical contract of a generated step along with the
	// HTTP handlers that expose it over the existing step protocol
	StepDef struct {
		Handler http.HandlerFunc
		ID      api.StepID
		Name    api.Name
		Type    api.StepType
		Inputs  []Attr
		Outputs []Attr
		Labels  api.Labels
	}

	// Attr is a named step input or output and its logical type
	Attr struct {
		Name     api.Name
		Type     api.AttributeType
		Optional bool
	}
)
