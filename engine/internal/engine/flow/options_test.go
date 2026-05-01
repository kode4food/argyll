package flow_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

func TestDefaultOptions(t *testing.T) {
	init := api.InitArgs{"a": {"b"}}
	meta := api.Metadata{"k": "v"}
	labels := api.Labels{"tier": "test"}
	parent := api.FlowStep{
		FlowID: "parent-flow",
		StepID: "parent-step",
	}
	tkn := api.Token("token-1")

	opts := flow.Defaults(
		flow.WithInit(init),
		flow.WithMetadata(meta),
		flow.WithParent(parent, tkn),
		flow.WithLabels(labels),
	)

	assert.Equal(t, init, opts.Init)
	assert.Equal(t, meta["k"], opts.Metadata["k"])
	assert.Equal(t, parent.FlowID, opts.Metadata[api.MetaParentFlowID])
	assert.Equal(t, parent.StepID, opts.Metadata[api.MetaParentStepID])
	assert.Equal(t, tkn, opts.Metadata[api.MetaParentWorkItemToken])
	assert.Equal(t, labels, opts.Labels)
}

func TestApply(t *testing.T) {
	opts := &flow.Options{}
	parent := api.FlowStep{
		FlowID: "parent-flow",
		StepID: "parent-step",
	}
	tkn := api.Token("token-1")
	init := api.InitArgs{"x": {"y"}}
	call.Apply(opts,
		flow.WithInit(init),
		flow.WithMetadata(api.Metadata{"m": "n"}),
		flow.WithParent(parent, tkn),
		flow.WithLabels(api.Labels{"l": "z"}),
	)

	assert.Equal(t, init, opts.Init)
	assert.Equal(t, "n", opts.Metadata["m"])
	assert.Equal(t, parent.FlowID, opts.Metadata[api.MetaParentFlowID])
	assert.Equal(t, parent.StepID, opts.Metadata[api.MetaParentStepID])
	assert.Equal(t, tkn, opts.Metadata[api.MetaParentWorkItemToken])
	assert.Equal(t, api.Labels{"l": "z"}, opts.Labels)
}
