package flow

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type (
	// Options contains optional parameters for starting a flow
	Options struct {
		Init       api.InitArgs
		Metadata   api.Metadata
		Tags       api.Tags
		Compensate bool
	}

	// Applier mutates *FlowOptions during StartFlow setup
	Applier = call.Applier[*Options]
)

// Defaults returns an Options instance with defaults applied
var Defaults = call.Defaults(func() *Options {
	return &Options{
		Init:     api.InitArgs{},
		Metadata: api.Metadata{},
		Tags:     api.Tags{},
	}
})

// WithInit sets the initial flow inputs
func WithInit(initState api.InitArgs) Applier {
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

// WithParent annotates flow metadata with the parent work item context
func WithParent(parent api.FlowStep, tkn api.Token) Applier {
	return func(opt *Options) {
		opt.Metadata = opt.Metadata.Apply(api.Metadata{
			api.MetaParentFlowID:        parent.FlowID,
			api.MetaParentStepID:        parent.StepID,
			api.MetaParentWorkItemToken: tkn,
		})
	}
}

// WithTags sets the flow tags
func WithTags(tags api.Tags) Applier {
	return func(opt *Options) {
		opt.Tags = tags
	}
}

// WithCompensate requests compensation of every succeeded work item in the
// flow if the flow fails
func WithCompensate(compensate bool) Applier {
	return func(opt *Options) {
		opt.Compensate = compensate
	}
}
