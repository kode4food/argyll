package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
		Endpoint    string
		Summary     string
		Description string
		Entity      string
		Inputs      []argSpec
		Outputs     []argSpec
	}

	argSpec struct {
		Name       string
		Service    string
		Type       string
		Location   string
		Path       string
		Schema     *SchemaFacts
		Confidence string
		Required   bool
	}

	opPair struct {
		Op     *openapi.Operation
		Method string
	}
)

const (
	locationHeader    = "header"
	openAPIHealthPath = "health"
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
	baseURL := resolveBaseURL(doc)
	for path, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		if isHealthPath(path) {
			continue
		}
		pathParams := collectParameters(item.Parameters, path)
		for _, pair := range operationsForPath(item) {
			entity := pathEntity(path)
			inputs := append(
				slices.Clone(pathParams),
				collectParameters(pair.Op.Parameters, path)...,
			)
			inputs = append(
				inputs,
				collectRequestBodyInputs(pair.Op, entity)...,
			)
			outputs := collectResponseOutputs(pair.Op, entity)
			id := operationID(pair.Op, pair.Method, path)
			summary := pair.Op.Summary
			if summary == "" {
				summary = humanizeID(id)
			}
			ops = append(ops, opSpec{
				ID:          id,
				Method:      pair.Method,
				Path:        path,
				Endpoint:    operationEndpoint(baseURL, path, inputs),
				Summary:     summary,
				Description: pair.Op.Description,
				Entity:      entity,
				Inputs:      dedupeArgs(inputs),
				Outputs:     dedupeArgs(outputs),
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

func operationEndpoint(baseURL, path string, inputs []argSpec) string {
	return joinURL(baseURL, pathWithQueryArgs(path, inputs))
}

func pathWithQueryArgs(path string, inputs []argSpec) string {
	params := queryArgs(inputs)
	if len(params) == 0 {
		return path
	}
	var b strings.Builder
	b.WriteString(path)
	if strings.Contains(path, "?") {
		b.WriteByte('&')
	} else {
		b.WriteByte('?')
	}
	for i, arg := range params {
		if i != 0 {
			b.WriteByte('&')
		}
		name := serviceNameForArg(arg)
		b.WriteString(url.QueryEscape(name))
		b.WriteString("={")
		b.WriteString(name)
		b.WriteByte('}')
	}
	return b.String()
}

func queryArgs(inputs []argSpec) []argSpec {
	var res []argSpec
	for _, arg := range dedupeArgs(inputs) {
		if arg.Location == "query" {
			res = append(res, arg)
		}
	}
	slices.SortFunc(res, func(a, b argSpec) int {
		return strings.Compare(serviceNameForArg(a), serviceNameForArg(b))
	})
	return res
}

func serviceNameForArg(arg argSpec) string {
	if arg.Service != "" {
		return arg.Service
	}
	return arg.Name
}

func isHealthPath(path string) bool {
	p := strings.Trim(strings.ToLower(path), "/")
	return p == openAPIHealthPath ||
		strings.HasSuffix(p, "/"+openAPIHealthPath)
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
	defaultEntity := pathEntity(path)
	for _, ref := range params {
		if ref == nil || ref.Value == nil {
			continue
		}
		p := ref.Value
		if shouldIgnoreParameter(p) {
			continue
		}
		entity := parameterEntity(path, p.Name)
		if entity == "" {
			entity = defaultEntity
		}
		res = append(res, argSpec{
			Name:       canonicalName(p.Name, entity),
			Service:    p.Name,
			Type:       schemaRefType(p.Schema),
			Schema:     schemaFacts(p.Schema),
			Required:   p.Required,
			Location:   p.In,
			Confidence: confidence(p.Name, entity),
		})
	}
	return res
}

func shouldIgnoreParameter(p *openapi.Parameter) bool {
	if p == nil {
		return true
	}
	return p.In == locationHeader &&
		strings.EqualFold(p.Name, api.HeaderWebhookURL)
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
			Schema:     schemaFacts(ref),
			Required:   required[name],
			Location:   "body",
			Confidence: confidence(name, entity),
		})
	}
	return res
}

func collectResponseOutputs(op *openapi.Operation, entity string) []argSpec {
	if op.Responses == nil {
		return collectCallbackOutputs(op, entity)
	}
	for code, ref := range op.Responses.Map() {
		if !strings.HasPrefix(code, "2") || ref == nil || ref.Value == nil {
			continue
		}
		schema := mediaSchema(ref.Value.Content)
		if schema == nil || schema.Value == nil {
			continue
		}
		return append(outputsFromSchema(schema, entity),
			collectCallbackOutputs(op, entity)...)
	}
	return collectCallbackOutputs(op, entity)
}

func collectCallbackOutputs(op *openapi.Operation, entity string) []argSpec {
	if op == nil || len(op.Callbacks) == 0 {
		return nil
	}
	var out []argSpec
	for _, ref := range op.Callbacks {
		if ref == nil || ref.Value == nil {
			continue
		}
		for _, item := range ref.Value.Map() {
			for _, pair := range operationsForPath(item) {
				body := pair.Op.RequestBody
				if body == nil || body.Value == nil {
					continue
				}
				schema := mediaSchema(body.Value.Content)
				if schema == nil || schema.Value == nil {
					continue
				}
				out = append(out, callbackOutputsFromSchema(schema, entity)...)
			}
		}
	}
	return out
}

func callbackOutputsFromSchema(
	ref *openapi.SchemaRef, entity string,
) []argSpec {
	if ref == nil || ref.Value == nil {
		return nil
	}
	props, _ := schemaProps(ref.Value)
	out := make([]argSpec, 0, len(props))
	for name, prop := range props {
		out = append(out, argSpec{
			Name:       canonicalName(name, entity),
			Service:    name,
			Type:       schemaRefType(prop),
			Location:   "callback",
			Confidence: confidence(name, entity),
		})
	}
	return out
}
