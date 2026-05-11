package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	openapi "github.com/getkin/kin-openapi/openapi3"
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
