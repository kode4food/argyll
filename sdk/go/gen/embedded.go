package gen

import (
	"context"
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	argyll "github.com/kode4food/argyll/sdk/go"
	"github.com/kode4food/argyll/sdk/go/convert"
)

// EmbedSync adapts a native function to a step handler an embedded engine runs in process,
// converting its values in memory
func EmbedSync[I, O any](
	in convert.Converter[I], out convert.Converter[O], fn func(I) (O, error),
) step.WorkFunc {
	return embedSync(in, out, func(_ api.Metadata, args I) (O, error) {
		return fn(args)
	})
}

// EmbedSyncCtx adapts an embedded function that accepts the Work Item's context
func EmbedSyncCtx[I, O any](
	in convert.Converter[I], out convert.Converter[O], fn func(argyll.StepContext, I) (O, error),
) step.WorkFunc {
	return embedSync(in, out, func(meta api.Metadata, args I) (O, error) {
		return fn(embedCtx(meta), args)
	})
}

// EmbedComp adapts a typed compensation function to a handler's in-process compensation
func EmbedComp[I any](in convert.Converter[I], fn func(I) error) step.WorkFunc {
	return embedComp(in, func(_ step.Runtime, _ api.Token, args I) error {
		return fn(args)
	})
}

// EmbedCompCtx adapts an embedded compensator that accepts the Work Item's context
func EmbedCompCtx[I any](
	in convert.Converter[I], fn func(argyll.StepContext, I) error,
) step.WorkFunc {
	return embedComp(in, func(rt step.Runtime, tkn api.Token, args I) error {
		return fn(embedCtx(embedMetadata(rt, tkn)), args)
	})
}

func embedSync[I, O any](
	in convert.Converter[I], out convert.Converter[O], fn func(api.Metadata, I) (O, error),
) step.WorkFunc {
	return func(
		rt step.Runtime, st *api.Step, inputs api.Args, tkn api.Token,
	) error {
		meta := embedMetadata(rt, tkn)
		args, err := in.From(embedArgs(st, inputs, meta))
		if err != nil {
			return errors.Join(ErrInvalidInputs, err)
		}
		res, err := invoke(func(in I) (O, error) { return fn(meta, in) }, args)
		if err != nil {
			return err
		}
		outputs, err := out.To(res)
		if err != nil {
			return err
		}
		return rt.CompleteWork(tkn, argsOf(outputs))
	}
}

func embedComp[I any](
	in convert.Converter[I], fn func(step.Runtime, api.Token, I) error,
) step.WorkFunc {
	return func(
		rt step.Runtime, _ *api.Step, args api.Args, tkn api.Token,
	) error {
		typed, err := in.From(membersOf(args))
		if err != nil {
			return errors.Join(ErrInvalidInputs, err)
		}
		_, err = invoke(
			func(in I) (struct{}, error) {
				return struct{}{}, fn(rt, tkn, in)
			},
			typed,
		)
		if err != nil {
			return err
		}
		return rt.CompleteWork(tkn, nil)
	}
}

func embedCtx(meta api.Metadata) argyll.StepContext {
	fid, _ := meta.GetString[api.FlowID](api.MetaFlowID)
	sid, _ := meta.GetString[api.StepID](api.MetaStepID)
	return argyll.NewStepContext(context.Background(), fid, sid, meta)
}

func embedMetadata(rt step.Runtime, tkn api.Token) api.Metadata {
	return rt.Metadata().Apply(api.Metadata{
		api.MetaFlowID:       rt.FlowID(),
		api.MetaStepID:       rt.StepID(),
		api.MetaReceiptToken: tkn,
	})
}

// embedArgs adds the meta attributes an engine would otherwise put in the request body
func embedArgs(st *api.Step, inputs api.Args, meta api.Metadata) map[string]any {
	res := membersOf(inputs)
	for name, attr := range st.Attributes {
		if !attr.IsMeta() {
			continue
		}
		if v, ok := meta[attr.MetaKey()]; ok {
			mapped, _ := st.MappedName(name)
			res[string(mapped)] = v
		}
	}
	return res
}

func membersOf(args api.Args) map[string]any {
	res := make(map[string]any, len(args))
	for name, v := range args {
		res[string(name)] = v
	}
	return res
}

func argsOf(v any) api.Args {
	members, _ := v.(map[string]any)
	res := make(api.Args, len(members))
	for name, member := range members {
		res[api.Name(name)] = member
	}
	return res
}
