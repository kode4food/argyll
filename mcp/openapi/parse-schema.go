package openapi

import (
	"maps"
	"slices"

	openapi "github.com/getkin/kin-openapi/openapi3"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var schemaTypeMap = map[string]string{
	"string":  "string",
	"number":  "number",
	"integer": "number",
	"boolean": "boolean",
	"object":  "object",
	"array":   "array",
	"null":    "null",
}

func outputsFromSchema(ref *openapi.SchemaRef, entity string) []argSpec {
	if ref == nil || ref.Value == nil {
		return nil
	}
	s := ref.Value
	var out []argSpec
	switch {
	case schemaIs(s, "object"):
		props, _ := schemaProps(s)
		if len(props) == 0 && entity != "" {
			out = append(out, argSpec{
				Name:       entity,
				Type:       "object",
				Location:   "response",
				Path:       "$",
				Confidence: "medium",
			})
		}
		for name, prop := range props {
			out = append(out, argSpec{
				Name:       canonicalName(name, ""),
				Service:    name,
				Type:       schemaRefType(prop),
				Location:   "response",
				Confidence: confidence(name, ""),
			})
		}
	case schemaIs(s, "array"):
		if entity != "" {
			out = append(out, argSpec{
				Name:       pluralName(entity),
				Type:       "array",
				Location:   "response",
				Path:       "$",
				Confidence: "medium",
			})
		}
	case len(s.AllOf) > 0:
		props, _ := schemaProps(s)
		for name, prop := range props {
			out = append(out, argSpec{
				Name:       canonicalName(name, entity),
				Service:    name,
				Type:       schemaRefType(prop),
				Location:   "response",
				Confidence: confidence(name, entity),
			})
		}
	}
	return out
}

func mediaSchema(content openapi.Content) *openapi.SchemaRef {
	if len(content) == 0 {
		return nil
	}
	if media, ok := content[api.JSONContentType]; ok && media != nil {
		return media.Schema
	}
	if media, ok := content["application/*+json"]; ok && media != nil {
		return media.Schema
	}
	for _, media := range content {
		if media != nil && media.Schema != nil {
			return media.Schema
		}
	}
	return nil
}

func schemaProps(
	schema *openapi.Schema,
) (map[string]*openapi.SchemaRef, map[string]bool) {
	props := map[string]*openapi.SchemaRef{}
	required := map[string]bool{}
	if schema == nil {
		return props, required
	}
	for _, name := range schema.Required {
		required[name] = true
	}
	maps.Copy(props, schema.Properties)
	for _, ref := range schema.AllOf {
		if ref == nil || ref.Value == nil {
			continue
		}
		partProps, partRequired := schemaProps(ref.Value)
		maps.Copy(props, partProps)
		for name := range partRequired {
			required[name] = true
		}
	}
	return props, required
}

func schemaRefType(ref *openapi.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return "any"
	}
	return schemaType(ref.Value)
}

func schemaEnum(ref *openapi.SchemaRef) []any {
	if ref == nil || ref.Value == nil || len(ref.Value.Enum) == 0 {
		return nil
	}
	return append([]any{}, ref.Value.Enum...)
}

func schemaFacts(ref *openapi.SchemaRef) *SchemaFacts {
	return schemaFactsDepth(ref, 0)
}

func schemaFactsDepth(ref *openapi.SchemaRef, depth int) *SchemaFacts {
	if ref == nil || ref.Value == nil {
		return nil
	}
	schema := ref.Value
	facts := &SchemaFacts{
		Type: schemaType(schema),
		Enum: schemaEnum(ref),
	}
	props, required := schemaProps(schema)
	if depth < 2 && len(props) != 0 {
		facts.Properties = make(map[string]SchemaFacts, len(props))
		for name, prop := range props {
			if child := schemaFactsDepth(prop, depth+1); child != nil {
				facts.Properties[name] = *child
			}
		}
		for name := range required {
			facts.Required = append(facts.Required, name)
		}
		slices.Sort(facts.Required)
	}
	if facts.Type == "any" && facts.Enum == nil &&
		len(facts.Properties) == 0 && len(facts.Required) == 0 {
		return nil
	}
	return facts
}

func schemaHasEnum(facts *SchemaFacts) bool {
	if facts == nil {
		return false
	}
	if len(facts.Enum) != 0 {
		return true
	}
	for _, prop := range facts.Properties {
		if schemaHasEnum(&prop) {
			return true
		}
	}
	return false
}

func schemaIs(schema *openapi.Schema, typ string) bool {
	if schema == nil || schema.Type == nil {
		return false
	}
	return schema.Type.Includes(typ)
}

func schemaType(schema *openapi.Schema) string {
	if schema == nil || schema.Type == nil {
		return "any"
	}
	for _, t := range schema.Type.Slice() {
		if mapped, ok := schemaTypeMap[t]; ok {
			return mapped
		}
	}
	return "any"
}
