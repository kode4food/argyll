import { AttributeRole, AttributeSpec, Step } from "@/app/api";
import { getSortedAttributes } from "@/utils/stepUtils";
import {
  Attribute,
  AttributeIndex,
  AttributeRoleType,
} from "./stepEditorTypes";

const ATTR_ROLE_TYPE: Record<AttributeRole, AttributeRoleType> = {
  [AttributeRole.Required]: "required",
  [AttributeRole.Optional]: "optional",
  [AttributeRole.Const]: "const",
  [AttributeRole.Meta]: "meta",
  [AttributeRole.Feed]: "feed",
  [AttributeRole.Output]: "output",
  [AttributeRole.Metric]: "metric",
};

const ATTR_ID_PREFIX: Record<AttributeRole, string> = {
  [AttributeRole.Required]: "required",
  [AttributeRole.Optional]: "optional",
  [AttributeRole.Const]: "const",
  [AttributeRole.Meta]: "meta",
  [AttributeRole.Feed]: "feed",
  [AttributeRole.Output]: "output",
  [AttributeRole.Metric]: "metric",
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
  const metricSelector =
    spec.role === AttributeRole.Metric ? spec.metric?.selector : undefined;

  return {
    id: `${prefix}-${index}-${timestamp}`,
    role: attrType,
    name,
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
    matchLanguage: inputConfig?.match?.language,
    matchScript: inputConfig?.match?.source,
    metaKey: spec.role === AttributeRole.Meta ? spec.meta?.key : undefined,
    feedId: spec.role === AttributeRole.Feed ? spec.feed?.id : undefined,
    metricLanguage: metricSelector?.language,
    metricScript: metricSelector?.source,
    mappingName: mappingConfig?.name,
    mappingLanguage: mappingConfig?.selector?.language,
    mappingScript: mappingConfig?.selector?.source,
    compensated: spec.compensated,
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
