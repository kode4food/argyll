package flowopt

import "github.com/kode4food/argyll/engine/pkg/api"

type (
	// Options contains optional parameters for starting a flow
	Options struct {
		Init     api.Args
		Metadata api.Metadata
		Labels   api.Labels
	}

	// Applier mutates Options during StartFlow setup
	Applier func(*Options)
)

// DefaultOptions returns an Options instance with defaults applied
func DefaultOptions(apps ...Applier) *Options {
	opt := &Options{
		Init:     api.Args{},
		Metadata: api.Metadata{},
		Labels:   api.Labels{},
	}
	ApplyOptions(opt, apps...)
	return opt
}

// ApplyOptions applies option appliers in order
func ApplyOptions(opt *Options, apps ...Applier) {
	for _, app := range apps {
		app(opt)
	}
}

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
