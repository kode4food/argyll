package engine

import (
	"encoding/json"
	"maps"

	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// computeWorkItems determines all work items that a step needs to execute
func computeWorkItems(step *api.Step, inputs api.Args) []api.Args {
	argNames := step.MultiArgNames()
	multiArgs := getMultiArgs(argNames, inputs)
	if len(multiArgs) == 0 {
		return []api.Args{inputs}
	}
	return cartesianProduct(multiArgs, inputs)
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
		return e.mapOutputAttributes(completed[0].Outputs, step)
	default:
		aggregated := collectWorkOutputs(completed, step)
		return e.mapOutputAttributes(aggregated, step)
	}
}

func (e *Engine) mapOutputAttributes(outputs api.Args, step *api.Step) api.Args {
	if step == nil {
		return outputs
	}

	res := api.Args{}
	for name, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}

		if attr.Mapping == "" {
			if value, ok := outputs[name]; ok {
				res[name] = value
			}
			continue
		}

		value, ok, err := e.mapper.MappingValue(attr.Mapping, outputs)
		if err != nil || !ok {
			continue
		}
		res[name] = value
	}
	return res
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
