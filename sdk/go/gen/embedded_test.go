package gen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/sdk/go/codec"
	"github.com/kode4food/argyll/sdk/go/gen"
)

type (
	testRuntime struct {
		meta    api.Metadata
		token   api.Token
		outputs api.Args
	}

	metaArgs struct {
		Token api.Token
		Flow  string
		Tier  string
	}
)

var _ step.Runtime = (*testRuntime)(nil)

func TestEmbeddedSyncOutputs(t *testing.T) {
	exec := gen.EmbeddedSync(sumArgsCodec(), sumResultCodec(),
		func(in sumArgs) (sumResult, error) {
			total := in.Left + in.Right
			return sumResult{Total: total, Doubled: total * 2}, nil
		})

	rt := &testRuntime{}
	err := exec(rt, &api.Step{}, api.Args{"left": 2.0, "right": 3.0}, "tkn")
	assert.NoError(t, err)
	assert.Equal(t, api.Token("tkn"), rt.token)
	assert.Equal(t, api.Args{"total": 5.0, "doubled": 10.0}, rt.outputs)
}

func TestEmbeddedSyncMeta(t *testing.T) {
	var got metaArgs
	exec := gen.EmbeddedSync(metaArgsCodec(), codec.Struct[struct{}](),
		func(in metaArgs) (struct{}, error) {
			got = in
			return struct{}{}, nil
		})
	st := &api.Step{
		Attributes: api.AttributeSpecs{
			"token": metaAttr(api.MetaReceiptToken),
			"flow":  metaAttr(api.MetaFlowID),
			"tier":  metaAttr("tier"),
			"extra": {Role: api.RoleRequired},
		},
	}

	rt := &testRuntime{meta: api.Metadata{"tier": "gold"}}
	assert.NoError(t, exec(rt, st, api.Args{}, "tkn"))
	assert.Equal(t, metaArgs{Token: "tkn", Flow: "flow-1", Tier: "gold"}, got)

	// a meta key the Flow never set leaves its field at the zero value
	rt = &testRuntime{}
	assert.NoError(t, exec(rt, st, api.Args{}, "tkn"))
	assert.Equal(t, metaArgs{Token: "tkn", Flow: "flow-1"}, got)
}

func TestEmbeddedSyncErrors(t *testing.T) {
	ok := func(sumArgs) (sumResult, error) {
		return sumResult{}, nil
	}
	rt := &testRuntime{}
	st := &api.Step{}

	exec := gen.EmbeddedSync(sumArgsCodec(), sumResultCodec(), ok)
	err := exec(rt, st, api.Args{"left": "x"}, "tkn")
	assert.ErrorIs(t, err, gen.ErrInvalidInputs)

	exec = gen.EmbeddedSync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, errRefused
		})
	assert.ErrorIs(t, exec(rt, st, api.Args{}, "tkn"), errRefused)

	exec = gen.EmbeddedSync(sumArgsCodec(), sumResultCodec(),
		func(sumArgs) (sumResult, error) {
			panic("boom")
		})
	var panicErr *gen.PanicError
	assert.ErrorAs(t, exec(rt, st, api.Args{}, "tkn"), &panicErr)

	exec = gen.EmbeddedSync(sumArgsCodec(), failCodec{}, ok)
	assert.ErrorIs(t, exec(rt, st, api.Args{}, "tkn"), errRefused)
	assert.Nil(t, rt.outputs)
}

func TestEmbeddedCompensate(t *testing.T) {
	var got compArgs
	comp := gen.EmbeddedCompensate(compArgsCodec(), func(in compArgs) error {
		got = in
		return nil
	})
	st := &api.Step{
		Attributes: api.AttributeSpecs{
			"left":  {Role: api.RoleRequired, Compensated: true},
			"right": {Role: api.RoleRequired},
			"total": {Role: api.RoleOutput, Compensated: true},
		},
	}

	done, err := comp(step.CompensateRequest{
		Step:    st,
		Inputs:  api.Args{"left": 2.0, "right": 3.0},
		Outputs: api.Args{"total": 5.0},
	})
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, compArgs{Left: 2, Total: 5}, got)
}

func TestEmbeddedCompensateErrors(t *testing.T) {
	st := &api.Step{
		Attributes: api.AttributeSpecs{
			"left": {Role: api.RoleRequired, Compensated: true},
		},
	}

	comp := gen.EmbeddedCompensate(compArgsCodec(), func(compArgs) error {
		return errRefused
	})
	done, err := comp(step.CompensateRequest{Step: st})
	assert.ErrorIs(t, err, errRefused)
	assert.False(t, done)

	done, err = comp(step.CompensateRequest{
		Step:   st,
		Inputs: api.Args{"left": "x"},
	})
	assert.ErrorIs(t, err, gen.ErrInvalidInputs)
	assert.False(t, done)
}

func TestEmbeddedSteps(t *testing.T) {
	steps, err := gen.EmbeddedSteps(`{"id":"sum","name":"Sum","type":"sum"}`)
	assert.NoError(t, err)
	assert.Equal(t, api.StepID("sum"), steps[0].ID)
	assert.Equal(t, api.StepType("sum"), steps[0].Type)

	_, err = gen.EmbeddedSteps(`{`)
	assert.Error(t, err)
}

func (r *testRuntime) FlowID() api.FlowID {
	return "flow-1"
}

func (r *testRuntime) StepID() api.StepID {
	return "step-1"
}

func (r *testRuntime) Metadata() api.Metadata {
	return r.meta
}

func (r *testRuntime) CompleteWork(tkn api.Token, outputs api.Args) error {
	r.token = tkn
	r.outputs = outputs
	return nil
}

func (r *testRuntime) UpdateHealth(api.HealthStatus, string) error {
	return nil
}

func metaAttr(key string) *api.AttributeSpec {
	return &api.AttributeSpec{
		Role: api.RoleMeta,
		Meta: &api.MetaConfig{Key: key},
	}
}

func metaArgsCodec() codec.Codec[metaArgs] {
	return codec.Struct(
		codec.Field("token", codec.Text[api.Token](),
			func(v *metaArgs) *api.Token {
				return &v.Token
			}),
		codec.Field("flow", codec.String, func(v *metaArgs) *string {
			return &v.Flow
		}),
		codec.Field("tier", codec.String, func(v *metaArgs) *string {
			return &v.Tier
		}),
	)
}
