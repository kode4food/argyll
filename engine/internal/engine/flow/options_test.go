package flow_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

func TestDefaultOptions(t *testing.T) {
	init := api.Args{"a": "b"}
	meta := api.Metadata{"k": "v"}
	labels := api.Labels{"tier": "test"}

	opts := flow.Defaults(
		flow.WithInit(init),
		flow.WithMetadata(meta),
		flow.WithLabels(labels),
	)

	assert.Equal(t, init, opts.Init)
	assert.Equal(t, meta, opts.Metadata)
	assert.Equal(t, labels, opts.Labels)
}

func TestApply(t *testing.T) {
	opts := &flow.Options{}
	call.Apply(opts,
		flow.WithInit(api.Args{"x": "y"}),
		flow.WithMetadata(api.Metadata{"m": "n"}),
		flow.WithLabels(api.Labels{"l": "z"}),
	)

	assert.Equal(t, api.Args{"x": "y"}, opts.Init)
	assert.Equal(t, api.Metadata{"m": "n"}, opts.Metadata)
	assert.Equal(t, api.Labels{"l": "z"}, opts.Labels)
}
