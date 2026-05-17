package mcp

type (
	semanticVerification struct {
		Status string          `json:"status"`
		Issues []semanticIssue `json:"issues,omitempty"`
	}

	semanticIssue struct {
		Expected any    `json:"expected,omitempty"`
		Actual   any    `json:"actual,omitempty"`
		StepID   string `json:"step_id"`
		Path     string `json:"path"`
		Kind     string `json:"kind"`
	}

	semanticConfig struct {
		Value any
		Path  string
		Kind  string
	}
)

func verifyAppliedSemantics(
	proposed []map[string]any, actual map[string]map[string]any,
) semanticVerification {
	var issues []semanticIssue
	for _, step := range proposed {
		id := stringValue(step["id"])
		if id == "" {
			continue
		}
		configs := semanticConfigs(step)
		if len(configs) == 0 {
			continue
		}
		readback, ok := actual[id]
		if !ok {
			issues = append(issues, semanticIssue{
				StepID: id,
				Path:   "$",
				Kind:   "missing_step",
			})
			continue
		}
		for _, cfg := range configs {
			got, ok := valueAtPath(readback, cfg.Path)
			if ok && sameJSON(got, cfg.Value) {
				continue
			}
			issues = append(issues, semanticIssue{
				StepID:   id,
				Path:     cfg.Path,
				Kind:     cfg.Kind,
				Expected: cfg.Value,
				Actual:   got,
			})
		}
	}
	if len(issues) != 0 {
		return semanticVerification{Status: "failed", Issues: issues}
	}
	return semanticVerification{Status: "passed"}
}

func semanticConfigs(step map[string]any) []semanticConfig {
	attrs, ok := asMap(step["attributes"])
	if !ok {
		return nil
	}
	var res []semanticConfig
	for name, raw := range attrs {
		attr, ok := asMap(raw)
		if !ok {
			continue
		}
		role := stringValue(attr["role"])
		switch role {
		case "required", "optional":
			path := "attributes." + name + "." + role
			res = appendRoleConfigs(res, path, attr[role])
		case "output":
			path := "attributes." + name + ".output"
			res = appendRoleConfigs(res, path, attr["output"])
		}
	}
	return res
}

func appendRoleConfigs(
	res []semanticConfig, base string, raw any,
) []semanticConfig {
	cfg, ok := asMap(raw)
	if !ok {
		return res
	}
	for _, key := range []string{"mapping", "match"} {
		value, ok := cfg[key]
		if !ok {
			continue
		}
		res = append(res, semanticConfig{
			Path:  base + "." + key,
			Kind:  key,
			Value: value,
		})
	}
	return res
}

func valueAtPath(root map[string]any, path string) (any, bool) {
	var curr any = root
	for _, part := range splitPath(path) {
		m, ok := asMap(curr)
		if !ok {
			return nil, false
		}
		curr, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return curr, true
}

func splitPath(path string) []string {
	var res []string
	start := 0
	for i, r := range path {
		if r != '.' {
			continue
		}
		res = append(res, path[start:i])
		start = i + 1
	}
	res = append(res, path[start:])
	return res
}
