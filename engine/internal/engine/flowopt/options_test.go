package flowopt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestDefaultOptions(t *testing.T) {
	init := api.Args{"a": "b"}
	meta := api.Metadata{"k": "v"}
	labels := api.Labels{"tier": "test"}

	opts := flowopt.DefaultOptions(
		flowopt.WithInit(init),
		flowopt.WithMetadata(meta),
		flowopt.WithLabels(labels),
	)

	assert.Equal(t, init, opts.Init)
	assert.Equal(t, meta, opts.Metadata)
	assert.Equal(t, labels, opts.Labels)
}

func TestApplyOptions(t *testing.T) {
	opts := &flowopt.Options{}
	flowopt.ApplyOptions(opts,
		flowopt.WithInit(api.Args{"x": "y"}),
		flowopt.WithMetadata(api.Metadata{"m": "n"}),
		flowopt.WithLabels(api.Labels{"l": "z"}),
	)

	assert.Equal(t, api.Args{"x": "y"}, opts.Init)
	assert.Equal(t, api.Metadata{"m": "n"}, opts.Metadata)
	assert.Equal(t, api.Labels{"l": "z"}, opts.Labels)
}
