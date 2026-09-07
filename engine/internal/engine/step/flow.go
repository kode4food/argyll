package step

import "github.com/kode4food/argyll/engine/pkg/api"

func flowHandler() *Handler {
	return &Handler{
		Children: flowChildren,
	}
}

func flowChildren(st *api.Step) []api.StepID {
	if st.Flow == nil {
		return nil
	}
	return st.Flow.Goals
}
