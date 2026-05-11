package openapi

import (
	"maps"

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
		if entity != "" {
			out = append(out, argSpec{
				Name:       entity,
				Type:       "object",
				Location:   "response",
				Path:       "$",
				Confidence: "medium",
			})
		}
		props, _ := schemaProps(s)
		for name, prop := range props {
			if !shouldExposeOutputProp(name, entity) {
				continue
			}
			out = append(out, argSpec{
				Name:       canonicalName(name, entity),
				Service:    name,
				Type:       schemaRefType(prop),
				Location:   "response",
				Confidence: confidence(name, entity),
			})
		}
		out = append(out, nestedOutputs(props, entity, "$")...)
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

func nestedOutputs(
	props map[string]*openapi.SchemaRef, entity, base string,
) []argSpec {
	var out []argSpec
	for name, prop := range props {
		if prop == nil || prop.Value == nil {
			continue
		}
		if !isWrapperProp(name) || !isObjectLike(prop.Value) {
			continue
		}
		nestedProps, _ := schemaProps(prop.Value)
		for childName, childProp := range nestedProps {
			if !shouldExposeOutputProp(childName, entity) {
				continue
			}
			out = append(out, argSpec{
				Name:       canonicalName(childName, entity),
				Type:       schemaRefType(childProp),
				Location:   "response",
				Path:       base + "." + name + "." + childName,
				Confidence: "medium",
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

func isObjectLike(schema *openapi.Schema) bool {
	return schemaIs(schema, "object") || len(schema.AllOf) > 0 ||
		len(schema.Properties) > 0
}

func schemaRefType(ref *openapi.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return "any"
	}
	return schemaType(ref.Value)
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
