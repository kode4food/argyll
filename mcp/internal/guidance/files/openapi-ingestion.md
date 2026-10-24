Use the Argyll MCP output below to ingest custom REST services into Argyll. Ingestion means turning external service operations into Argyll step registrations that the planner can use.

Your responsibility is to infer the business flow represented by the operations, existing steps, names, descriptions, schemas, examples, enum values, and request/response shapes.

Start with `analyze_openapi_contract` to extract neutral OpenAPI facts. The MCP does not generate final step registrations from OpenAPI; you author those registrations from the contract facts.

## Step JSON schema

The schema used to validate registrations is in `engine-api.yaml` (embedded alongside this file). The definitions that matter most for ingestion are `StepRegistration`, `AttributeSpec`, `RequiredConfig`, `OptionalConfig`, `OutputConfig`, `MappingConfig`, `ScriptConfig`, and `HttpConfig`.

Get these structural rules wrong and the engine either rejects the step or silently drops the field:

- Every step **must** have `id`. Steps missing `id` are silently skipped by `diff_proposed_steps` and `apply_proposed_steps` with no error.
- Valid `type` values: `service`, `script`, `flow`.
- The `http` object nests the call under `invoke`, which uses the key `endpoint` (not `url`).
- Both `http.invoke` and `http.compensate` accept `mode: sync` or `mode: async`; each defaults to sync independently.
- If the service exposes a health check endpoint, set `http.health` to that URL. The engine uses it to track step availability, and always polls it with `GET`.
- If the service can undo the call, add `http.compensate` with its own `endpoint` and optional `method`. Its presence is what makes the step compensatable.

```json
{
  "id": "my-step",
  "name": "My Step",
  "type": "service",
  "http": {
    "invoke": {
      "endpoint": "http://host/path",
      "method": "POST",
      "mode": "sync"
    },
    "compensate": {
      "endpoint": "http://host/path/{id}",
      "method": "DELETE",
      "mode": "async"
    },
    "health": "http://host/health"
  },
  "attributes": { }
}
```

## Attribute spec

`attributes` is a JSON **object** (map keyed by attribute name), not an array. Valid roles: `required`, `optional`, `const`, `output`.

Mapping and match configuration nest **inside a sub-object keyed by the role name** on the attribute, not at the top level of the attribute. Both `mapping` and `match` are objects, not strings:

- `mapping` → `{ "name": "service_field_name" }` (or add `"script"` for a transform)
- `match` → `{ "language": "argyll", "script": "value == \"email\"" }` - a `ScriptConfig` object, never a bare string
- `argyll` is the language identifier for Argyll Expressions in mappings, predicates, matches, and Space selectors. Script Steps use Lua. Predicates receive named Step inputs. Matches and selectors receive `value`. Predicates using Argyll Expressions must return a boolean

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
user_data:
  role: output
  type: object
  output:
    mapping:
      script:
        language: argyll
        script: "user_data.user"
```

For service outputs `{"user": {"id": "u-123", "name": "Ada"}}`, this output mapping stores `{"id": "u-123", "name": "Ada"}` as `user_data`. Output mappings receive the complete outputs object under the output Attribute's name.

For an input conversion, an Attribute `price_dollars` with value `12.5` can use `mapping: {name: amount, script: {language: argyll, script: "amount * 100"}}` to send `amount: 1250` to the service. Input mappings receive the individual value under the mapped parameter name.

In JSON, an attribute with both a match discriminator and a field rename looks like:

```json
"channel": {
  "type": "string",
  "role": "required",
  "required": {
    "match":   { "language": "argyll", "script": "value == \"email\"" },
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

When a true discriminator exists, use `required.match` to make otherwise similar operations mutually exclusive. For `required.match` using Argyll Expressions, Argyll evaluates the candidate Attribute value as `value` and requires a boolean result.

Use these generic Argyll Expressions patterns:

- Scalar value equals a discriminator: `value == "some-value"`
- Object field equals a discriminator: `value?.field_name == "some-value"`
- Multiple acceptable scalar values: `value in ["a", "b"]`
- Multiple required object conditions: `value?.kind == "a" && value?.region == "b"`

Only create the match on the attribute that carries the discriminator. Do not add required matches to every enum-bearing input.

Keep each step's required inputs minimal. A step should declare only the values that endpoint actually needs, plus discriminator attributes needed for planning.

Treat missing inputs as graph diagnostics. Do not invent bridge steps or synthetic services to hide missing business mappings.

## Workflow

Before applying registrations, inspect the contract operations and existing steps, author refined Argyll step definitions, then use diff/apply tools and preview a plan.

Do not apply the raw OpenAPI extraction when service field names appear in required attributes used by the planner. First produce refined Argyll step definitions with consistent planner attributes, role-specific `mapping`, and any inferred `required.match` discriminators.

After applying registrations, inspect the `verification` block returned by `apply_proposed_steps`. If it reports missing or changed `mapping` or `match` paths, treat the registration as semantically failed even if the engine accepted the request.

After verification of the saved registration passes, use `preview_plan` with representative goal step IDs (not attribute names). The `goals` field takes a chain of fallback goal sets of step IDs, such as `{"steps": ["step-a", "step-b"], "else": {"steps": ["step-c"]}}`. If service-specific request fields appear as missing initial inputs, refine the step attributes and mappings before running the flow.

{{payload}}
