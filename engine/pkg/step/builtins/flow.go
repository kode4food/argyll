package builtins

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

// Flow returns the standard nested-flow handler
func Flow() *step.Handler {
	return &step.Handler{
		Children: func(st *api.Step) []api.StepID {
			if st.Flow == nil {
				return nil
			}
			return st.Flow.Goals
		},
	}
}
