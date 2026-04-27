package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	openapi "github.com/getkin/kin-openapi/openapi3"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	opSpec struct {
		ID          string
		Method      string
		Path        string
		Summary     string
		Description string
		Entity      string
		Inputs      []argSpec
		Outputs     []argSpec
		Step        map[string]any
	}

	argSpec struct {
		Name       string
		Service    string
		Type       string
		Required   bool
		Location   string
		Path       string
		Confidence string
	}

	opPair struct {
		Method string
		Op     *openapi.Operation
	}
)

var (
	ErrInvalidParams = errors.New("invalid params")
)

func parseDoc(args Args) (*openapi.T, error) {
	data, err := specBytes(args)
	if err != nil {
		return nil, err
	}

	loader := openapi.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	return doc, nil
}

func specBytes(args Args) ([]byte, error) {
	if args.Spec != nil {
		data, err := json.Marshal(args.Spec)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
		}
		return data, nil
	}
	if strings.TrimSpace(args.SpecText) == "" {
		return nil, fmt.Errorf(
			"%w: spec_text or spec is required",
			ErrInvalidParams,
		)
	}
	return []byte(args.SpecText), nil
}

func collectOperations(doc *openapi.T) []opSpec {
	if doc.Paths == nil {
		return nil
	}

	var ops []opSpec
	for path, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		pathParams := collectParameters(item.Parameters, path)
		for _, pair := range operationsForPath(item) {
			entity := inferEntity(path)
			inputs := append(
				slices.Clone(pathParams),
				collectParameters(pair.Op.Parameters, path)...,
			)
			inputs = append(
				inputs,
				collectRequestBodyInputs(pair.Op, entity)...,
			)
			outputs := collectResponseOutputs(pair.Op, entity)
			id := inferOperationID(pair.Op, pair.Method, path)
			summary := pair.Op.Summary
			if summary == "" {
				summary = humanizeID(id)
			}
			step := map[string]any{
				"id":   id,
				"name": summary,
				"type": "sync",
				"labels": map[string]any{
					"argyll.source":       "openapi",
					"argyll.operation_id": pair.Op.OperationID,
					"argyll.method":       pair.Method,
					"argyll.path":         path,
				},
				"http": map[string]any{
					"method":   pair.Method,
					"endpoint": joinURL(resolveBaseURL(doc), path),
				},
				"attributes": buildAttributes(inputs, outputs),
			}
			ops = append(ops, opSpec{
				ID:          id,
				Method:      pair.Method,
				Path:        path,
				Summary:     summary,
				Description: pair.Op.Description,
				Entity:      entity,
				Inputs:      dedupeArgs(inputs),
				Outputs:     dedupeArgs(outputs),
				Step:        step,
			})
		}
	}

	slices.SortFunc(ops, func(a, b opSpec) int {
		if cmp := strings.Compare(a.Path, b.Path); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Method, b.Method)
	})
	return ops
}

func operationsForPath(item *openapi.PathItem) []opPair {
	var res []opPair
	if item.Get != nil {
		res = append(res, opPair{Method: http.MethodGet, Op: item.Get})
	}
	if item.Post != nil {
		res = append(res, opPair{Method: http.MethodPost, Op: item.Post})
	}
	if item.Put != nil {
		res = append(res, opPair{Method: http.MethodPut, Op: item.Put})
	}
	if item.Delete != nil {
		res = append(res, opPair{Method: http.MethodDelete, Op: item.Delete})
	}
	return res
}

func collectParameters(params openapi.Parameters, path string) []argSpec {
	var res []argSpec
	pathEntity := inferEntity(path)
	for _, ref := range params {
		if ref == nil || ref.Value == nil {
			continue
		}
		p := ref.Value
		entity := inferParamEntity(path, p.Name)
		if entity == "" {
			entity = pathEntity
		}
		res = append(res, argSpec{
			Name:       canonicalName(p.Name, entity),
			Service:    p.Name,
			Type:       schemaRefType(p.Schema),
			Required:   p.Required,
			Location:   p.In,
			Confidence: confidence(p.Name, entity),
		})
	}
	return res
}

func collectRequestBodyInputs(op *openapi.Operation, entity string) []argSpec {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	schema := mediaSchema(op.RequestBody.Value.Content)
	if schema == nil || schema.Value == nil {
		return nil
	}
	props, required := schemaProps(schema.Value)
	res := make([]argSpec, 0, len(props))
	for name, ref := range props {
		res = append(res, argSpec{
			Name:       canonicalName(name, entity),
			Service:    name,
			Type:       schemaRefType(ref),
			Required:   required[name],
			Location:   "body",
			Confidence: confidence(name, entity),
		})
	}
	return res
}

func collectResponseOutputs(op *openapi.Operation, entity string) []argSpec {
	if op.Responses == nil {
		return nil
	}
	for code, ref := range op.Responses.Map() {
		if !strings.HasPrefix(code, "2") || ref == nil || ref.Value == nil {
			continue
		}
		schema := mediaSchema(ref.Value.Content)
		if schema == nil || schema.Value == nil {
			continue
		}
		return outputsFromSchema(schema, entity)
	}
	return nil
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
		out = append(out, nestedOutputsFromObjectProps(props, entity, "$")...)
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

func nestedOutputsFromObjectProps(
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

func schemaRefType(ref *openapi.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return "any"
	}
	return schemaType(ref.Value)
}

func schemaType(schema *openapi.Schema) string {
	if schema == nil {
		return "any"
	}
	switch {
	case schemaIs(schema, "string"):
		return "string"
	case schemaIs(schema, "number"), schemaIs(schema, "integer"):
		return "number"
	case schemaIs(schema, "boolean"):
		return "boolean"
	case schemaIs(schema, "object"):
		return "object"
	case schemaIs(schema, "array"):
		return "array"
	case schemaIs(schema, "null"):
		return "null"
	default:
		return "any"
	}
}

func isObjectLike(schema *openapi.Schema) bool {
	return schemaIs(schema, "object") || len(schema.AllOf) > 0 ||
		len(schema.Properties) > 0
}

func schemaIs(schema *openapi.Schema, typ string) bool {
	if schema == nil || schema.Type == nil {
		return false
	}
	return schema.Type.Includes(typ)
}
