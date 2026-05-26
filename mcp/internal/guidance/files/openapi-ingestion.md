Use the Argyll MCP output below to ingest custom REST services into Argyll. Ingestion means turning external service operations into Argyll step registrations that the planner can use.

Your responsibility is to infer the business flow represented by the operations, existing steps, names, descriptions, schemas, examples, enum values, and request/response shapes.

Start with `analyze_openapi_contract` to extract neutral OpenAPI facts. The MCP does not generate final step registrations from OpenAPI; you author those registrations from the contract facts.

## Step JSON schema

The schema used to validate registrations is in `engine-api.yaml` (embedded alongside this file). The definitions that matter most for ingestion are `StepRegistration`, `AttributeSpec`, `RequiredConfig`, `OptionalConfig`, `OutputConfig`, `MappingConfig`, `ScriptConfig`, and `HttpConfig`.

Key structural rules — get these wrong and the engine either rejects the step or silently drops the field:

- Every step **must** have `id`. Steps missing `id` are silently skipped by `diff_proposed_steps` and `apply_proposed_steps` with no error.
- Valid `type` values: `sync`, `async`, `script`, `flow`.
- The `http` object uses the key `endpoint` (not `url`).
- If the service exposes a health check endpoint, set `http.health_check` to that URL. The engine uses it to track step availability.

```json
{
  "id": "my-step",
  "name": "My Step",
  "type": "sync",
  "http": {
    "endpoint": "http://host/path",
    "method": "POST",
    "health_check": "http://host/health"
  },
  "attributes": { }
}
```

## Attribute spec

`attributes` is a JSON **object** (map keyed by attribute name), not an array. Valid roles: `required`, `optional`, `const`, `output`.

Mapping and match configuration nest **inside a sub-object keyed by the role name** on the attribute — not at the top level of the attribute. Both `mapping` and `match` are objects, not strings:

- `mapping` → `{ "name": "service_field_name" }` (or add `"script"` for a transform)
- `match` → `{ "language": "ale", "script": "..." }` - a `ScriptConfig` object, never a bare string
- `jpath` is Argyll's language identifier for JSONPath-style query expressions; it has Argyll-specific behavior and is intended for mappings and predicates, not executable Script Steps

The spec's own example (from `StepRegistration.attributes`):

```yaml
input_text:
  role: required
  type: string
user_email:
  role: required
  type: string
  required:
    mapping:
      name: email          # renames planner attr → service field
result:
  role: output
  type: object
  output:
    mapping:
      script:
        language: jpath
        script: "$.response.data"   # extracts from nested response
```

In JSON, an attribute with both a match discriminator and a field rename looks like:

```json
"channel": {
  "type": "string",
  "role": "required",
  "required": {
    "match":   { "language": "ale", "script": "(eq value \"email\")" },
    "mapping": { "name": "service_channel_field" }
  }
}
```

Omit the role sub-object entirely when no mapping or match is needed.

## Mapping and match guidance

Do not assume identical field names are required across services. If two service-specific fields represent the same business value, choose one planner attribute name for the flow and keep each service field name in the `mapping.name` for that step attribute.

Do not add mappings where Argyll's default pass-through already works. If the planner attribute and service field name are the same, omit mapping.

Use `required.mapping` or `optional.mapping` for service input renames. Use `output.mapping` only when an output field must be renamed or extracted from a JSON path.

The MCP may report enum values as schema facts. Do not treat every narrowed enum as a required match. Use the operation names, descriptions, shapes, enum values, and surrounding business flow to decide whether an attribute is a true discriminator.

When a true discriminator exists, use `required.match` to make otherwise similar operations mutually exclusive. For Ale `required.match`, Argyll evaluates the candidate attribute value as `value`.

Use these generic Ale patterns:

- Scalar value equals a discriminator: `(eq value "some-value")`
- Object field equals a discriminator: `(eq (:field_name value) "some-value")`
- Multiple acceptable scalar values: `(or (eq value "a") (eq value "b"))`
- Multiple required object conditions: `(and (eq (:kind value) "a") (eq (:region value) "b"))`

Only create the match on the attribute that carries the discriminator. Do not add required matches to every enum-bearing input.

Keep each step's required inputs minimal. A step should declare only the values that endpoint actually needs, plus discriminator attributes needed for planning.

Treat missing inputs as graph diagnostics. Do not invent bridge steps or synthetic services to hide missing business mappings.

## Workflow

Before applying registrations, inspect the contract operations and existing steps, author refined Argyll step definitions, then use diff/apply tools and preview a plan.

Do not apply the raw OpenAPI extraction when service field names appear in required attributes used by the planner. First produce refined Argyll step definitions with consistent planner attributes, role-specific `mapping`, and any inferred `required.match` discriminators.

After applying registrations, inspect the `verification` block returned by `apply_proposed_steps`. If it reports missing or changed `mapping` or `match` paths, treat the registration as semantically failed even if the engine accepted the request.

After verification of the saved registration passes, use `preview_plan` with representative goal step IDs (not attribute names). The `goals` field takes step IDs. If service-specific request fields appear as missing initial inputs, refine the step attributes and mappings before running the flow.

{{payload}}
