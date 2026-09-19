package gen

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/sdk/go/codec"
)

// EmbeddedSync adapts a plain Go function to a step handler an embedded engine
// runs in process, converting its values without serializing them
func EmbeddedSync[I, O any](
	in codec.Codec[I], out codec.Codec[O], fn func(I) (O, error),
) step.InvokeFunc {
	return func(
		rt step.Runtime, st *api.Step, inputs api.Args, tkn api.Token,
	) error {
		args, err := in.FromValue(embeddedArgs(rt, st, inputs, tkn))
		if err != nil {
			return errors.Join(ErrInvalidInputs, err)
		}
		res, err := invoke(fn, args)
		if err != nil {
			return err
		}
		outputs, err := out.ToValue(res)
		if err != nil {
			return err
		}
		return rt.CompleteWork(tkn, argsOf(outputs))
	}
}

// EmbeddedCompensate adapts a typed compensation function to a step handler's
// in-process compensation
func EmbeddedCompensate[I any](
	in codec.Codec[I], fn func(I) error,
) step.CompensateFunc {
	return func(req step.CompensateRequest) (bool, error) {
		args, err := in.FromValue(compensationArgs(req))
		if err != nil {
			return false, errors.Join(ErrInvalidInputs, err)
		}
		_, err = invoke(
			func(in I) (struct{}, error) {
				return struct{}{}, fn(in)
			},
			args,
		)
		return err == nil, err
	}
}

// EmbeddedSteps decodes the specifications of steps an embedded engine runs
func EmbeddedSteps(specs ...string) ([]*api.Step, error) {
	res := make([]*api.Step, len(specs))
	for i, spec := range specs {
		var st api.Step
		if err := json.Unmarshal([]byte(spec), &st); err != nil {
			return nil, fmt.Errorf("%w: %s", err, spec)
		}
		res[i] = &st
	}
	return res, nil
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
	res := make(map[string]any, len(inputs))
	for name, v := range inputs {
		res[string(name)] = v
	}
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

// compensationArgs selects the compensated attributes, from the inputs or the
// outputs they belong to, that a compensation request carries
func compensationArgs(req step.CompensateRequest) map[string]any {
	res := map[string]any{}
	for name, attr := range req.Step.Attributes {
		if attr == nil || !attr.Compensated {
			continue
		}
		mapped, _ := req.Step.MappedName(name)
		src := req.Inputs
		if attr.IsOutput() {
			src = req.Outputs
		}
		if v, ok := src[mapped]; ok {
			res[string(mapped)] = v
		}
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
