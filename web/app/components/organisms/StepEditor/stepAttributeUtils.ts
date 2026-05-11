import {
  AttributeRole,
  AttributeSpec,
  AttributeType,
  MappingConfig,
  OptionalConfig,
  RequiredConfig,
  SCRIPT_LANGUAGE_JPATH,
  ScriptConfig,
  Step,
} from "@/app/api";
import { getArgIcon } from "@/utils/iconRegistry";
import { getSortedAttributes } from "@/utils/stepUtils";
import {
  Attribute,
  AttributeIndex,
  AttributeRoleType,
} from "./stepEditorTypes";

const ATTR_ROLE_TYPE: Record<AttributeRole, AttributeRoleType> = {
  [AttributeRole.Required]: "input",
  [AttributeRole.Optional]: "optional",
  [AttributeRole.Const]: "const",
  [AttributeRole.Output]: "output",
};

const ATTR_ID_PREFIX: Record<AttributeRole, string> = {
  [AttributeRole.Required]: "input",
  [AttributeRole.Optional]: "input",
  [AttributeRole.Const]: "const",
  [AttributeRole.Output]: "output",
};

function buildSingleAttribute(
  name: string,
  spec: AttributeSpec,
  idx: AttributeIndex
): Attribute {
  const { index, timestamp } = idx;
  const attrType = ATTR_ROLE_TYPE[spec.role];
  const prefix = ATTR_ID_PREFIX[spec.role];
  const inputConfig =
    spec.role === AttributeRole.Required
      ? spec.required
      : spec.role === AttributeRole.Optional
        ? spec.optional
        : undefined;
  const mappingConfig =
    spec.role === AttributeRole.Output
      ? spec.output?.mapping
      : inputConfig?.mapping;

  return {
    id: `${prefix}-${index}-${timestamp}`,
    attrType,
    name,
    dataType: spec.type || AttributeType.String,
    defaultValue:
      spec.role === AttributeRole.Optional && spec.optional?.default
        ? spec.optional.default
        : spec.role === AttributeRole.Const && spec.const?.value
          ? spec.const.value
          : undefined,
    deadline:
      spec.role === AttributeRole.Optional && spec.optional?.deadline
        ? spec.optional.deadline
        : undefined,
    collect: inputConfig?.collect || "first",
    forEach: inputConfig?.for_each || false,
    matchLanguage:
      spec.role === AttributeRole.Required
        ? spec.required?.match?.language
        : undefined,
    matchScript:
      spec.role === AttributeRole.Required
        ? spec.required?.match?.script
        : undefined,
    mappingName: mappingConfig?.name,
    mappingLanguage: mappingConfig?.script?.language,
    mappingScript: mappingConfig?.script?.script,
  };
}

export function buildAttributesFromStep(step: Step | null): Attribute[] {
  if (!step) return [];
  const timestamp = Date.now();
  return getSortedAttributes(step.attributes || {}).map(
    ({ name, spec }, index) =>
      buildSingleAttribute(name, spec, { index, timestamp })
  );
}

export function getAttributeIconProps(attrType: AttributeRoleType) {
  const argType = attrType === "input" ? "required" : attrType;
  return getArgIcon(argType);
}

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
  if (a.defaultValue?.trim()) optional.default = a.defaultValue.trim();
  if (a.deadline) optional.deadline = a.deadline;
  if (mapping) optional.mapping = mapping;
  return Object.keys(optional).length > 0 ? optional : undefined;
}

function buildMatchConfig(a: Attribute): ScriptConfig {
  return {
    language: a.matchLanguage?.trim() || SCRIPT_LANGUAGE_JPATH,
    script: a.matchScript?.trim() || "",
  };
}

function buildMappingConfig(a: Attribute): MappingConfig | undefined {
  const mappingName = a.mappingName?.trim();
  const mappingScript = a.mappingScript?.trim();
  if (!mappingName && !mappingScript) return undefined;
  const config: MappingConfig = {};
  if (mappingName) config.name = mappingName;
  if (mappingScript) {
    config.script = {
      language: a.mappingLanguage?.trim() || "lua",
      script: mappingScript,
    };
  }
  return config;
}

const ROLE_MAP: Record<AttributeRoleType, AttributeRole> = {
  input: AttributeRole.Required,
  optional: AttributeRole.Optional,
  const: AttributeRole.Const,
  output: AttributeRole.Output,
};

export function createStepAttributes(
  attributes: Attribute[]
): Record<string, AttributeSpec> {
  const stepAttributes: Record<string, AttributeSpec> = {};
  attributes.forEach((a) => {
    const role = ROLE_MAP[a.attrType];
    const spec: AttributeSpec = { role, type: a.dataType };
    const mapping = buildMappingConfig(a);

    if (a.attrType === "input") {
      const required = buildInputAttrSpec(a, mapping);
      if (required) spec.required = required;
    } else if (a.attrType === "optional") {
      const optional = buildOptionalAttrSpec(a, mapping);
      if (optional) spec.optional = optional;
    } else if (a.attrType === "const") {
      if (a.defaultValue?.trim()) spec.const = { value: a.defaultValue.trim() };
    } else if (a.attrType === "output") {
      if (mapping) spec.output = { mapping };
    }

    stepAttributes[a.name] = spec;
  });
  return stepAttributes;
}
