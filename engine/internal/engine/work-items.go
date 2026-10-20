package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

const MaxWorkItemsPerStep = 10000

var (
	ErrTooManyWorkItems = errors.New("too many work items")
)

func (e *Engine) collectStepOutputs(
	items api.WorkItems, st *api.Step,
) api.Args {
	completed := make([]api.WorkState, 0, len(items))
	for _, work := range items {
		if policy.WorkSucceeded(work.Status) {
			completed = append(completed, work)
		}
	}

	switch len(completed) {
	case 0:
		return nil
	case 1:
		return e.mapper.MapOutputs(st, completed[0].Outputs)
	default:
		aggregated := collectWorkOutputs(completed, st)
		return e.mapper.MapOutputs(st, aggregated)
	}
}

func computeWorkItems(st *api.Step, inputs api.Args) ([]api.Args, error) {
	argNames := st.MultiArgNames()
	multiArgs := getMultiArgs(argNames, inputs)
	if len(multiArgs) == 0 {
		return []api.Args{{}}, nil
	}

	n, err := productSize(multiArgs)
	if err != nil || n == 0 {
		return nil, err
	}
	return cartesianProduct(multiArgs), nil
}

func getMultiArgs(argNames []api.Name, inputs api.Args) MultiArgs {
	multiArgs := MultiArgs{}

	for _, name := range argNames {
		if arr, ok := inputs[name].([]any); ok {
			multiArgs[name] = arr
		}
	}

	return multiArgs
}

// productSize bounds non-empty products before multiplication can overflow
func productSize(multiArgs MultiArgs) (int, error) {
	for _, arr := range multiArgs {
		if len(arr) == 0 {
			return 0, nil
		}
	}
	n := 1
	for _, arr := range multiArgs {
		if n > MaxWorkItemsPerStep/len(arr) {
			return 0, fmt.Errorf("%w: max %d",
				ErrTooManyWorkItems, MaxWorkItemsPerStep)
		}
		n *= len(arr)
	}
	return n, nil
}

func cartesianProduct(multiArgs MultiArgs) []api.Args {
	if len(multiArgs) == 0 {
		return nil
	}

	names, arrays := extractMultiArgs(multiArgs)

	var result []api.Args
	var generate func(int, api.Args)
	generate = func(depth int, current api.Args) {
		if depth == len(arrays) {
			result = append(result, current)
			return
		}

		name := names[depth]
		for _, val := range arrays[depth] {
			next := current.Set(name, val)
			generate(depth+1, next)
		}
	}

	generate(0, nil)
	return result
}

func extractMultiArgs(multiArgs MultiArgs) ([]api.Name, [][]any) {
	var names []api.Name
	var arrays [][]any
	for name, arr := range multiArgs {
		names = append(names, name)
		arrays = append(arrays, arr)
	}
	return names, arrays
}

func collectWorkOutputs(completed []api.WorkState, st *api.Step) api.Args {
	aggregated := map[api.Name][]map[string]any{}
	var multiArgNames []api.Name
	if st != nil {
		multiArgNames = st.MultiArgNames()
	}

	for _, item := range completed {
		for name, value := range item.Outputs {
			entry := map[string]any{}
			for _, argName := range multiArgNames {
				if val, ok := item.Inputs[argName]; ok {
					entry[string(argName)] = val
				}
			}
			entry[string(name)] = value

			aggregated[name] = append(aggregated[name], entry)
		}
	}

	outputs := api.Args{}
	for name, values := range aggregated {
		outputs[name] = values
	}
	return outputs
}
