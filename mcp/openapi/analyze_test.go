package openapi_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/mcp/openapi"
)

const catalogSpec = `
openapi: 3.0.3
info:
  title: Catalog
  version: "2"
servers:
  - url: https://catalog.example/api/
paths:
  /v1/categories/by-name/{name}:
    get:
      parameters:
        - name: name
          in: path
          required: true
          schema: {type: string}
      responses:
        "200":
          description: found
          content:
            application/json:
              schema: {type: object}
  /v2/addresses/{id}:
    get:
      operationId: getAddress
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      responses:
        "200":
          description: found
          content:
            application/vnd.catalog+json:
              schema:
                allOf:
                  - type: object
                    properties:
                      accountId: {type: string}
                  - type: object
                    properties:
                      status: {type: string, enum: [open, closed]}
  /companies/search:
    get:
      operationId: searchCompanies
      responses:
        "200":
          description: found
          content:
            text/plain:
              schema:
                type: array
                items: {type: object}
  /inventory:
    post:
      operationId: createInventory
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [sku]
              properties:
                sku: {type: string}
                notes: {}
                shipment:
                  type: object
                  required: [carrier]
                  properties:
                    carrier: {type: string, enum: [ups, dhl]}
      responses:
        "201":
          description: created
`

func TestAnalyzeContractNamesAndSchemas(t *testing.T) {
	res, err := openapi.AnalyzeContract(openapi.Args{SpecText: catalogSpec}, nil, nil)
	if !assert.NoError(t, err) {
		return
	}
	ops := map[string]openapi.Operation{}
	for _, op := range res.Operations {
		ops[op.ID] = op
	}

	// No operationId, so the ID comes from the method and path. Versions, by- segments, and
	// plurals are dropped from the entity
	byName := ops["get-v1-categories-by-name-name"]
	assert.Equal(t, "category", byName.Entity)
	assert.Equal(t, "https://catalog.example/api/v1/categories/by-name/{name}", byName.Endpoint)
	assert.Equal(t, []openapi.Arg{{
		Name:       "category",
		Type:       "object",
		Location:   "response",
		Confidence: "medium",
		Path:       "$",
	}}, byName.Outputs)

	// A path id names its entity, and allOf parts merge, from a vendor JSON media type
	address := ops["get-address"]
	assert.Equal(t, "address", address.Entity)
	assert.Equal(t, "address_id", address.Inputs[0].Name)
	assert.Equal(t, "high", address.Inputs[0].Confidence)
	assert.Equal(t, "number", address.Inputs[0].Type)
	assert.Equal(t, []string{"account_id", "status"}, argNames(address.Outputs))
	assert.Equal(t, "accountId", address.Outputs[0].Service)

	// An action word is skipped, and an array from any media type is the plural entity
	search := ops["search-companies"]
	assert.Equal(t, "company", search.Entity)
	assert.Equal(t, []string{"companies"}, argNames(search.Outputs))
	assert.Equal(t, "array", search.Outputs[0].Type)

	// Request bodies keep nested schema facts, including enums and required fields
	inventory := ops["create-inventory"]
	assert.Equal(t, []string{"notes", "shipment", "sku"}, argNames(inventory.Inputs))
	assert.Equal(t, "any", inventory.Inputs[0].Type)
	assert.Nil(t, inventory.Inputs[0].Schema)
	assert.Equal(t, &openapi.SchemaFacts{
		Type:     "object",
		Required: []string{"carrier"},
		Properties: map[string]openapi.SchemaFacts{
			"carrier": {
				Type: "string",
				Enum: []any{"ups", "dhl"},
			},
		},
	}, inventory.Inputs[1].Schema)
	assert.True(t, inventory.Inputs[2].Required)
}

const ordersSpec = `
openapi: 3.0.3
info:
  title: Orders
  version: "1"
servers:
  - url: https://{env}.orders.example/
    variables:
      env: {default: api}
paths:
  /health:
    get:
      responses:
        "200": {description: ok}
  /orders/{orderId}:
    parameters:
      - name: orderId
        in: path
        required: true
        schema: {type: string}
    put:
      operationId: updateOrder
      parameters:
        - name: notify
          in: query
          schema: {type: boolean}
        - name: Argyll-Webhook-URL
          in: header
          schema: {type: string}
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                status: {type: string, enum: [open, closed]}
      responses:
        "404": {description: missing}
    delete:
      operationId: deleteOrder
      responses:
        "204": {description: deleted}
  /orders:
    post:
      operationId: placeOrder
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                callback: {type: string}
      callbacks:
        done:
          "{$request.body#/callback}":
            post:
              requestBody:
                content:
                  application/json:
                    schema:
                      type: object
                      properties:
                        receiptId: {type: string}
              responses:
                "200": {description: ok}
      responses:
        "202": {description: accepted}
  /v1/{code}:
    get:
      operationId: lookup
      parameters:
        - name: code
          in: path
          required: true
          schema: {type: string}
      responses:
        "200": {description: found}
`

func TestAnalyzeContractOperations(t *testing.T) {
	existing := []openapi.Step{{
		ID:       "place-order",
		Name:     "Place Order",
		Source:   "catalog",
		Required: []string{"callback"},
	}}
	res, err := openapi.AnalyzeContract(
		openapi.Args{SpecText: ordersSpec}, existing, []string{"partial spec"},
	)
	if !assert.NoError(t, err) {
		return
	}
	ops := map[string]openapi.Operation{}
	for _, op := range res.Operations {
		ops[op.ID] = op
	}

	assert.Equal(t, "https://api.orders.example", res.BaseURL)
	assert.Equal(t, []string{"partial spec"}, res.Warnings)
	assert.Equal(t, "place-order", res.ExistingSteps[0].ID)
	assert.NotContains(t, ops, "get-health")

	// Path parameters join the operation's own, query parameters join the endpoint, and the
	// engine's webhook header is not an input
	update := ops["update-order"]
	assert.Equal(t, "PUT", update.Method)
	assert.Equal(t,
		"https://api.orders.example/orders/{orderId}?notify={notify}", update.Endpoint)
	assert.Equal(t, []string{"notify", "order_id", "status"}, argNames(update.Inputs))
	assert.Empty(t, update.Outputs)
	assert.Equal(t, "DELETE", ops["delete-order"].Method)

	// An async operation's callback body supplies its outputs
	place := ops["place-order"]
	assert.Equal(t, []string{"receipt_id"}, argNames(place.Outputs))

	// Without an entity, names fall back to the service field and are flagged
	lookup := ops["lookup"]
	assert.Empty(t, lookup.Entity)
	assert.Equal(t, "low", lookup.Inputs[0].Confidence)
	assert.Contains(t, res.Ambiguities,
		"could not confidently identify a canonical entity from path /v1/{code}")
	assert.Contains(t, res.Ambiguities,
		"low-confidence canonical input name for service field code")
}

func TestAnalyzeContractPrefersEnumArgs(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: Dupes, version: "1"}
paths:
  /customers:
    get:
      operationId: listCustomers
      parameters:
        - {name: customerId, in: query, schema: {}}
        - {name: customer_id, in: query, schema: {type: string}}
        - name: customer-id
          in: query
          schema:
            type: object
            properties:
              kind: {type: string, enum: [person]}
      responses:
        "200": {description: ok}
`
	res, err := openapi.AnalyzeContract(openapi.Args{SpecText: spec}, nil, nil)
	if !assert.NoError(t, err) {
		return
	}
	inputs := res.Operations[0].Inputs
	assert.Equal(t, []string{"customer_id"}, argNames(inputs))
	assert.Equal(t, "customer-id", inputs[0].Service)
}

const edgeSpec = `
openapi: 3.0.3
info: {title: Edges, version: "1"}
paths:
  /search?mode=full:
    get:
      operationId: search
      parameters:
        - {name: term, in: query, schema: {type: string}}
        - {name: page, in: query, schema: {type: integer}}
      responses:
        "200": {description: ok}
  /accounts/v1/{id}:
    get:
      operationId: getAccount
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200": {description: ok}
  /tickets/search/{id}:
    get:
      operationId: findTicket
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200": {description: ok}
  /{id}:
    get:
      operationId: getAny
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {name: orderid, in: query, schema: {type: string}}
      responses:
        "200": {description: ok}
  /statuses:
    get:
      operationId: listStatuses
      responses:
        "200":
          description: ok
          content:
            application/*+json:
              schema:
                type: array
                items: {type: string}
  /notes:
    post:
      operationId: addNote
      requestBody:
        content:
          text/plain: {}
      callbacks:
        empty:
          "{$request.body#/callback}":
            post:
              responses:
                "200": {description: ok}
        bare:
          "{$request.body#/other}":
            post:
              requestBody:
                content:
                  text/plain: {}
              responses:
                "200": {description: ok}
      responses:
        "202": {description: accepted}
  /parts:
    post:
      operationId: addPart
      requestBody:
        content:
          application/json:
            schema:
              allOf:
                - type: object
                  required: [part_id]
                  properties:
                    part_id: {type: string}
      responses:
        "201": {description: created}
`

func TestAnalyzeContractEdges(t *testing.T) {
	res, err := openapi.AnalyzeContract(openapi.Args{SpecText: edgeSpec}, nil, nil)
	if !assert.NoError(t, err) {
		return
	}
	ops := map[string]openapi.Operation{}
	for _, op := range res.Operations {
		ops[op.ID] = op
	}

	// Query arguments sort by name and extend a path that already has a query
	assert.Equal(t, "/search?mode=full&page={page}&term={term}", ops["search"].Endpoint)

	// Version and action segments are skipped when naming an id's entity
	assert.Equal(t, "account_id", ops["get-account"].Inputs[0].Name)
	assert.Equal(t, "ticket_id", ops["find-ticket"].Inputs[0].Name)
	assert.Equal(t, []string{"id", "order_id"}, argNames(ops["get-any"].Inputs))

	assert.Equal(t, []string{"status"}, argNames(ops["list-statuses"].Outputs))
	assert.Empty(t, ops["add-note"].Inputs)
	assert.Empty(t, ops["add-note"].Outputs)
	assert.True(t, ops["add-part"].Inputs[0].Required)
}

func TestAnalyzeContractSpecMap(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Mapped",
			"version": "1",
		},
		"paths": map[string]any{},
	}
	res, err := openapi.AnalyzeContract(openapi.Args{Spec: spec}, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Mapped", res.Info.Title)
	assert.Empty(t, res.BaseURL)
}

func TestAnalyzeContractRejects(t *testing.T) {
	for _, spec := range []string{
		"",
		"openapi: [",
		"openapi: 3.0.3\npaths: {}\n",
	} {
		_, err := openapi.AnalyzeContract(openapi.Args{SpecText: spec}, nil, nil)
		assert.ErrorIs(t, err, openapi.ErrInvalidParams)
	}

	spec := map[string]any{"openapi": math.Inf(1)}
	_, err := openapi.AnalyzeContract(openapi.Args{Spec: spec}, nil, nil)
	assert.ErrorIs(t, err, openapi.ErrInvalidParams)
}

func argNames(args []openapi.Arg) []string {
	res := make([]string, len(args))
	for i, a := range args {
		res[i] = a.Name
	}
	return res
}
