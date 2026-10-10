package gen

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/sdk/go/convert"
)

// EmbeddedSync adapts a native function to a step handler an embedded engine
// runs in process, converting its values in memory
func EmbeddedSync[I, O any](
	in convert.Converter[I], out convert.Converter[O], fn func(I) (O, error),
) step.InvokeFunc {
	return func(
		rt step.Runtime, st *api.Step, inputs api.Args, tkn api.Token,
	) error {
		args, err := in.From(embeddedArgs(rt, st, inputs, tkn))
		if err != nil {
			return errors.Join(ErrInvalidInputs, err)
		}
		res, err := invoke(fn, args)
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

// EmbeddedCompensate adapts a typed compensation function to a step handler's
// in-process compensation
func EmbeddedCompensate[I any](
	in convert.Converter[I], fn func(I) error,
) step.CompensateFunc {
	return func(
		rt step.Runtime, _ *api.Step, args api.Args, tkn api.Token,
	) error {
		typed, err := in.From(membersOf(args))
		if err != nil {
			return errors.Join(ErrInvalidInputs, err)
		}
		_, err = invoke(
			func(in I) (struct{}, error) {
				return struct{}{}, fn(in)
			},
			typed,
		)
		if err != nil {
			return err
		}
		return rt.CompleteWork(tkn, nil)
	}
}

// embeddedArgs adds the meta attributes an engine would otherwise put in the
// request body, drawn from the runtime and the work item's token
func embeddedArgs(
	rt step.Runtime, st *api.Step, inputs api.Args, tkn api.Token,
) map[string]any {
	meta := rt.Metadata().Apply(api.Metadata{
		api.MetaFlowID:       rt.FlowID(),
		api.MetaStepID:       rt.StepID(),
		api.MetaReceiptToken: tkn,
	})
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
