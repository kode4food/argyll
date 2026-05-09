import {
  Step,
  AttributeRole,
  AttributeSpec,
  AttributeType,
  InputCollect,
} from "@/app/api";
import { STEP_TYPE_ORDER } from "@/app/constants";
import {
  IconArrayMultiple,
  IconAttributeMatch,
  IconDuration,
  IconMapping,
  type LucideIcon,
} from "@/utils/iconRegistry";

export type StepType = "resolver" | "processor" | "collector" | "standalone";

export interface OrderedAttribute {
  name: string;
  spec: AttributeSpec;
}

export const getSortedAttributes = (
  attributes: Record<string, AttributeSpec>
): OrderedAttribute[] => {
  const sortedByName = Object.entries(attributes)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, spec]) => ({ name, spec }));

  return [
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Required),
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Const),
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Optional),
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Output),
  ];
};

const getJsonType = (parsed: any): string => {
  if (parsed === null) return "null";
  if (Array.isArray(parsed)) return "array";
  return typeof parsed;
};

export const validateDefaultValue = (
  value: string,
  type: AttributeType
): {
  valid: boolean;
  errorKey?: string;
  errorVars?: Record<string, string>;
} => {
  if (!value.trim()) {
    return { valid: true };
  }

  const trimmed = value.trim();

  let parsed: any;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { valid: false, errorKey: "validation.jsonInvalid" };
  }

  if (type === AttributeType.Any) {
    return { valid: true };
  }

  const jsonType = getJsonType(parsed);

  switch (type) {
    case AttributeType.String:
      if (jsonType !== "string") {
        return { valid: false, errorKey: "validation.jsonString" };
      }
      break;

    case AttributeType.Number:
      if (jsonType !== "number") {
        return { valid: false, errorKey: "validation.jsonNumber" };
      }
      break;

    case AttributeType.Boolean:
      if (jsonType !== "boolean") {
        return { valid: false, errorKey: "validation.jsonBoolean" };
      }
      break;

    case AttributeType.Object:
      if (jsonType !== "object") {
        return { valid: false, errorKey: "validation.jsonObject" };
      }
      break;

    case AttributeType.Array:
      if (jsonType !== "array") {
        return { valid: false, errorKey: "validation.jsonArray" };
      }
      break;

    case AttributeType.Null:
      if (jsonType !== "null") {
        return { valid: false, errorKey: "validation.jsonNull" };
      }
      break;
  }

  return { valid: true };
};

export const getStepType = (step: Step): StepType => {
  if (!step.attributes) return "standalone";

  const hasRequiredInputs = Object.values(step.attributes).some(
    (attr) => attr.role === AttributeRole.Required
  );
  const hasOutputs = Object.values(step.attributes).some(
    (attr) => attr.role === AttributeRole.Output
  );

  if (hasOutputs && !hasRequiredInputs) return "resolver";
  if (!hasOutputs && hasRequiredInputs) return "collector";
  if (hasOutputs && hasRequiredInputs) return "processor";
  return "standalone";
};

export type AttributeModifier =
  | { kind: "icon"; Icon: LucideIcon }
  | { kind: "collect"; collect: InputCollect };

export const getAttributeModifiers = (
  spec: AttributeSpec
): AttributeModifier[] => {
  const modifiers: AttributeModifier[] = [];
  if (spec.optional?.deadline) {
    modifiers.push({ kind: "icon", Icon: IconDuration });
  }
  if (spec.required?.match) {
    modifiers.push({ kind: "icon", Icon: IconAttributeMatch });
  }
  const config = spec.required ?? spec.optional ?? spec.output;
  if (config?.mapping) {
    modifiers.push({ kind: "icon", Icon: IconMapping });
  }
  const collect = spec.required?.collect ?? spec.optional?.collect;
  if (collect && collect !== "first") {
    modifiers.push({ kind: "collect", collect });
  }
  if (spec.required?.for_each || spec.optional?.for_each) {
    modifiers.push({ kind: "icon", Icon: IconArrayMultiple });
  }
  return modifiers;
};

const collectTitleKeyMap: Record<InputCollect, string> = {
  first: "",
  last: "attribute.modifierCollectLast",
  all: "attribute.modifierCollectAll",
  some: "attribute.modifierCollectSome",
  none: "attribute.modifierCollectNone",
};

export const getModifierTitleKey = (modifier: AttributeModifier): string => {
  if (modifier.kind === "collect") {
    return collectTitleKeyMap[modifier.collect];
  }
  if (modifier.Icon === IconDuration) return "attribute.modifierDeadline";
  if (modifier.Icon === IconAttributeMatch) return "attribute.modifierMatch";
  if (modifier.Icon === IconMapping) return "attribute.modifierMapping";
  if (modifier.Icon === IconArrayMultiple) return "attribute.modifierForEach";
  return "";
};

export const sortStepsByType = (steps: Step[]): Step[] => {
  return [...steps].sort((a, b) => {
    const aType = getStepType(a);
    const bType = getStepType(b);
    const orderDiff = STEP_TYPE_ORDER[aType] - STEP_TYPE_ORDER[bType];
    if (orderDiff !== 0) return orderDiff;
    return a.name.localeCompare(b.name);
  });
};
