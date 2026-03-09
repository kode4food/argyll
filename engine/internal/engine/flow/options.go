package flow

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type (
	// Options contains optional parameters for starting a flow
	Options struct {
		Init     api.Args
		Metadata api.Metadata
		Labels   api.Labels
	}

	// Applier mutates *FlowOptions during StartFlow setup
	Applier = call.Applier[*Options]
)

// Defaults returns an Options instance with defaults applied
var Defaults = call.Defaults(func() *Options {
	return &Options{
		Init:     api.Args{},
		Metadata: api.Metadata{},
		Labels:   api.Labels{},
	}
})

// WithInit sets the initial flow inputs
func WithInit(initState api.Args) Applier {
	return func(opt *Options) {
		opt.Init = initState
	}
}

// WithMetadata sets the flow metadata
func WithMetadata(meta api.Metadata) Applier {
	return func(opt *Options) {
		opt.Metadata = meta
	}
}

// WithLabels sets the flow labels
func WithLabels(labels api.Labels) Applier {
	return func(opt *Options) {
		opt.Labels = labels
	}
}
