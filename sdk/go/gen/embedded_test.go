package gen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/sdk/go/convert"
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

	failConverter struct{}
)

var _ step.Runtime = (*testRuntime)(nil)

func TestEmbeddedSyncOutputs(t *testing.T) {
	exec := gen.EmbeddedSync(sumArgsConverter(), sumResultConverter(),
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
	exec := gen.EmbeddedSync(metaArgsConverter(), convert.Struct[struct{}](),
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

	exec := gen.EmbeddedSync(sumArgsConverter(), sumResultConverter(), ok)
	err := exec(rt, st, api.Args{"left": "x"}, "tkn")
	assert.ErrorIs(t, err, gen.ErrInvalidInputs)

	exec = gen.EmbeddedSync(sumArgsConverter(), sumResultConverter(),
		func(sumArgs) (sumResult, error) {
			return sumResult{}, errRefused
		})
	assert.ErrorIs(t, exec(rt, st, api.Args{}, "tkn"), errRefused)

	exec = gen.EmbeddedSync(sumArgsConverter(), sumResultConverter(),
		func(sumArgs) (sumResult, error) {
			panic("boom")
		})
	var panicErr *gen.PanicError
	assert.ErrorAs(t, exec(rt, st, api.Args{}, "tkn"), &panicErr)

	exec = gen.EmbeddedSync(sumArgsConverter(), failConverter{}, ok)
	assert.ErrorIs(t, exec(rt, st, api.Args{}, "tkn"), errRefused)
	assert.Nil(t, rt.outputs)
}

func TestEmbeddedCompensate(t *testing.T) {
	var got compArgs
	comp := gen.EmbeddedCompensate(compArgsConverter(),
		func(in compArgs) error {
			got = in
			return nil
		})
	rt := &testRuntime{}
	args := api.Args{"left": 2.0, "total": 5.0}
	assert.NoError(t, comp(rt, &api.Step{}, args, "tkn"))
	assert.Equal(t, api.Token("tkn"), rt.token)
	assert.Equal(t, compArgs{Left: 2, Total: 5}, got)
}

func TestEmbeddedCompensateErrors(t *testing.T) {
	comp := gen.EmbeddedCompensate(compArgsConverter(),
		func(compArgs) error {
			return errRefused
		})
	rt := &testRuntime{}
	err := comp(rt, &api.Step{}, api.Args{}, "tkn")
	assert.ErrorIs(t, err, errRefused)
	assert.Empty(t, rt.token)

	err = comp(rt, &api.Step{}, api.Args{"left": "x"}, "tkn")
	assert.ErrorIs(t, err, gen.ErrInvalidInputs)
	assert.Empty(t, rt.token)
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

func (failConverter) From(any) (sumResult, error) {
	return sumResult{}, errRefused
}

func (failConverter) To(sumResult) (any, error) {
	return nil, errRefused
}

func metaAttr(key string) *api.AttributeSpec {
	return &api.AttributeSpec{
		Role: api.RoleMeta,
		Meta: &api.MetaConfig{Key: key},
	}
}

func metaArgsConverter() convert.Converter[metaArgs] {
	return convert.Struct(
		convert.Field("token", convert.Text[api.Token](),
			func(v *metaArgs) *api.Token {
				return &v.Token
			}),
		convert.Field("flow", convert.Text[string](),
			func(v *metaArgs) *string {
				return &v.Flow
			}),
		convert.Field("tier", convert.Text[string](),
			func(v *metaArgs) *string {
				return &v.Tier
			}),
	)
}

func sumArgsConverter() convert.Converter[sumArgs] {
	return convert.Struct(
		convert.Field("left", convert.Number[int](), func(v *sumArgs) *int {
			return &v.Left
		}),
		convert.Field("right", convert.Number[int](),
			func(v *sumArgs) *int {
				return &v.Right
			}),
	)
}

func sumResultConverter() convert.Converter[sumResult] {
	return convert.Struct(
		convert.Field("total", convert.Number[int](),
			func(v *sumResult) *int {
				return &v.Total
			}),
		convert.Field("doubled", convert.Number[int](),
			func(v *sumResult) *int {
				return &v.Doubled
			}),
	)
}

func compArgsConverter() convert.Converter[compArgs] {
	return convert.Struct(
		convert.Field("left", convert.Number[int](), func(v *compArgs) *int {
			return &v.Left
		}),
		convert.Field("total", convert.Number[int](),
			func(v *compArgs) *int {
				return &v.Total
			}),
	)
}
