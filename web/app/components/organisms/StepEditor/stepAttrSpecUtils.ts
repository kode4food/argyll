import {
  AttributeRole,
  AttributeSpec,
  MappingConfig,
  OptionalConfig,
  RequiredConfig,
  SCRIPT_LANGUAGE_ARGYLL,
  ScriptConfig,
} from "@/app/api";
import { Attribute, AttributeRoleType } from "./stepEditorTypes";

function buildInputAttrSpec(
  a: Attribute,
  mapping: MappingConfig | undefined
): RequiredConfig | undefined {
  const required: RequiredConfig = {};
  if (a.collect && a.collect !== "first") required.collect = a.collect;
  if (a.forEach) required.for_each = true;
  if (a.matchScript?.trim()) required.match = buildMatchConfig(a);
  if (mapping) required.mapping = mapping;
  return Object.keys(required).length > 0 ? required : undefined;
}

function buildOptionalAttrSpec(
  a: Attribute,
  mapping: MappingConfig | undefined
): OptionalConfig | undefined {
  const optional: OptionalConfig = {};
  if (a.collect && a.collect !== "first") optional.collect = a.collect;
  if (a.forEach) optional.for_each = true;
  if (a.matchScript?.trim()) optional.match = buildMatchConfig(a);
  if (a.defaultValue?.trim()) optional.default = a.defaultValue.trim();
  if (a.deadline) optional.deadline = a.deadline;
  if (mapping) optional.mapping = mapping;
  return Object.keys(optional).length > 0 ? optional : undefined;
}

function buildMatchConfig(a: Attribute): ScriptConfig {
  return {
    language: a.matchLanguage?.trim() || SCRIPT_LANGUAGE_ARGYLL,
    source: a.matchScript?.trim() || "",
  };
}

function buildMappingConfig(a: Attribute): MappingConfig | undefined {
  const mappingName = a.mappingName?.trim();
  const mappingScript = a.mappingScript?.trim();
  if (!mappingName && !mappingScript) return undefined;
  const config: MappingConfig = {};
  if (mappingName) config.name = mappingName;
  if (mappingScript) {
    config.selector = {
      language: a.mappingLanguage?.trim() || SCRIPT_LANGUAGE_ARGYLL,
      source: mappingScript,
    };
  }
  return config;
}

const ROLE_MAP: Record<AttributeRoleType, AttributeRole> = {
  required: AttributeRole.Required,
  optional: AttributeRole.Optional,
  const: AttributeRole.Const,
  meta: AttributeRole.Meta,
  feed: AttributeRole.Feed,
  output: AttributeRole.Output,
  metric: AttributeRole.Metric,
};

export function createStepAttributes(
  attributes: Attribute[]
): Record<string, AttributeSpec> {
  const stepAttributes: Record<string, AttributeSpec> = {};
  attributes.forEach((a) => {
    const role = ROLE_MAP[a.role];
    const spec: AttributeSpec = { role };
    if (a.compensated) spec.compensated = true;
    const mapping = buildMappingConfig(a);

    if (a.role === "required") {
      const required = buildInputAttrSpec(a, mapping);
      if (required) spec.required = required;
    } else if (a.role === "optional") {
      const optional = buildOptionalAttrSpec(a, mapping);
      if (optional) spec.optional = optional;
    } else if (a.role === "const") {
      if (a.defaultValue?.trim()) spec.const = { value: a.defaultValue.trim() };
    } else if (a.role === "meta") {
      if (a.metaKey?.trim()) spec.meta = { key: a.metaKey.trim() };
    } else if (a.role === "feed") {
      if (a.feedId?.trim()) spec.feed = { id: a.feedId.trim() };
    } else if (a.role === "output") {
      if (mapping) spec.output = { mapping };
    } else if (a.role === "metric") {
      const source = a.metricScript?.trim();
      if (source) {
        spec.metric = {
          selector: {
            language: a.metricLanguage?.trim() || SCRIPT_LANGUAGE_ARGYLL,
            source,
          },
        };
      }
    }

    stepAttributes[a.name] = spec;
  });
  return stepAttributes;
}
