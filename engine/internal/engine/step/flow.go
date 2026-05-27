package step

import "github.com/kode4food/argyll/engine/pkg/api"

// RegisterFlowHandler registers the built-in handler for flow steps
func RegisterFlowHandler(r *Registry) {
	r.Register(api.StepTypeFlow, Handler{
		Execute:  executeFlow,
		Children: flowChildren,
	})
}

func executeFlow(
	rt Runtime, _ *api.Step, inputs api.Args, tkn api.Token,
) error {
	init := api.InitArgs{}
	for name, value := range inputs {
		init[name] = []any{value}
	}
	_, err := rt.StartChildFlow(tkn, init)
	return err
}

func flowChildren(st *api.Step) []api.StepID {
	if st.Flow == nil {
		return nil
	}
	return st.Flow.Goals
}
