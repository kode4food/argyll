package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const MaxWorkItemsPerStep = 10000

var (
	ErrTooManyWorkItems = errors.New("too many work items")
)

func (e *Engine) collectStepOutputs(
	items api.WorkItems, step *api.Step,
) api.Args {
	completed := make([]*api.WorkState, 0, len(items))
	for _, item := range items {
		if item.Status == api.WorkSucceeded {
			completed = append(completed, item)
		}
	}

	switch len(completed) {
	case 0:
		return nil
	case 1:
		return e.mapper.MapOutputs(step, completed[0].Outputs)
	default:
		aggregated := collectWorkOutputs(completed, step)
		return e.mapper.MapOutputs(step, aggregated)
	}
}

func computeWorkItems(step *api.Step, inputs api.Args) ([]api.Args, error) {
	argNames := step.MultiArgNames()
	multiArgs := getMultiArgs(argNames, inputs)
	if len(multiArgs) == 0 {
		return []api.Args{inputs}, nil
	}

	if n := productSize(multiArgs); n > MaxWorkItemsPerStep {
		return nil, fmt.Errorf("%w: %d (max %d)",
			ErrTooManyWorkItems, n, MaxWorkItemsPerStep)
	}
	return cartesianProduct(multiArgs, inputs), nil
}

func getMultiArgs(argNames []api.Name, inputs api.Args) MultiArgs {
	multiArgs := MultiArgs{}

	for _, name := range argNames {
		if arr := asArray(inputs[name]); arr != nil {
			multiArgs[name] = arr
		}
	}

	return multiArgs
}

func asArray(value any) []any {
	if value == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	result := gjson.ParseBytes(jsonBytes)
	if !result.IsArray() {
		return nil
	}

	arr := make([]any, 0, len(result.Array()))
	for _, item := range result.Array() {
		arr = append(arr, item.Value())
	}
	return arr
}

func productSize(multiArgs MultiArgs) int {
	n := 1
	for _, arr := range multiArgs {
		n *= len(arr)
	}
	return n
}

func cartesianProduct(multiArgs MultiArgs, baseInputs api.Args) []api.Args {
	if len(multiArgs) == 0 {
		return nil
	}

	names, arrays := extractMultiArgs(multiArgs)

	var result []api.Args
	var generate func(int, api.Args)
	generate = func(depth int, current api.Args) {
		if depth == len(arrays) {
			result = append(result,
				combineInputs(baseInputs, current, multiArgs),
			)
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

func combineInputs(baseInputs, current api.Args, multiArgs MultiArgs) api.Args {
	inputs := api.Args{}
	for k, v := range baseInputs {
		if _, isMulti := multiArgs[k]; !isMulti {
			inputs[k] = v
		}
	}
	maps.Copy(inputs, current)
	return inputs
}

func collectWorkOutputs(completed []*api.WorkState, step *api.Step) api.Args {
	aggregated := map[api.Name][]map[string]any{}
	var multiArgNames []api.Name
	if step != nil {
		multiArgNames = step.MultiArgNames()
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
