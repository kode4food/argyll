package mcp

import (
	"embed"

	"gopkg.in/yaml.v3"
)

type (
	apiSpec struct {
		Name    string
		Title   string
		Version string
		Doc     map[string]any
	}

	specEntry struct {
		Name string
		File string
	}
)

//go:embed specs/*.yaml
var specFS embed.FS

var specEntries = []specEntry{
	{
		Name: "engine",
		File: "specs/engine-api.yaml",
	},
	{
		Name: "step-interface",
		File: "specs/step-interface.yaml",
	},
}

func loadAllSpecs() ([]*apiSpec, error) {
	specs := make([]*apiSpec, 0, len(specEntries))
	for _, entry := range specEntries {
		raw, err := specFS.ReadFile(entry.File)
		if err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		spec := &apiSpec{
			Name: entry.Name,
			Doc:  doc,
		}
		if info, ok := doc["info"].(map[string]any); ok {
			if title, ok := info["title"].(string); ok {
				spec.Title = title
			}
			if version, ok := info["version"].(string); ok {
				spec.Version = version
			}
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func compactOpenAPI(doc map[string]any) map[string]any {
	compact := map[string]any{}
	if info, ok := doc["info"]; ok {
		compact["info"] = info
	}
	if paths, ok := doc["paths"]; ok {
		compact["paths"] = paths
	}
	if components, ok := doc["components"]; ok {
		compact["components"] = components
	}
	if tags, ok := doc["tags"]; ok {
		compact["tags"] = tags
	}
	return compact
}
