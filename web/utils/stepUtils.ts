import {
  Step,
  AttributeRole,
  AttributeSpec,
  AttributeType,
  InputCollect,
  ScriptConfig,
} from "@/app/api";
import { STEP_TYPE_ORDER } from "@/app/constants";
import {
  IconArrayMultiple,
  IconAttributeMatch,
  IconDuration,
  IconMapping,
  type ArgType,
  type LucideIcon,
} from "@/utils/iconRegistry";

export type StepType = "resolver" | "processor" | "collector" | "standalone";

export interface OrderedAttribute {
  name: string;
  spec: AttributeSpec;
}

export const ROLE_ARG_TYPE: Record<AttributeRole, ArgType> = {
  required: "required",
  optional: "optional",
  const: "const",
  meta: "meta",
  output: "output",
};

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
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Meta),
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Output),
  ];
};

const getJsonType = (parsed: any): string => {
  if (parsed === null) return "null";
  if (Array.isArray(parsed)) return "array";
  return typeof parsed;
};

type ValidationResult = {
  valid: boolean;
  errorKey?: string;
  errorVars?: Record<string, string>;
};

export const validateDefaultValue = (
  value: string,
  type: AttributeType
): ValidationResult => {
  if (!value.trim()) {
    return { valid: true };
  }

  let parsed: any;
  try {
    parsed = JSON.parse(value.trim());
  } catch {
    return { valid: false, errorKey: "validation.jsonInvalid" };
  }

  const constraints: Partial<
    Record<AttributeType, { jsonType: string; errorKey: string }>
  > = {
    [AttributeType.String]: {
      jsonType: "string",
      errorKey: "validation.jsonString",
    },
    [AttributeType.Number]: {
      jsonType: "number",
      errorKey: "validation.jsonNumber",
    },
    [AttributeType.Boolean]: {
      jsonType: "boolean",
      errorKey: "validation.jsonBoolean",
    },
    [AttributeType.Object]: {
      jsonType: "object",
      errorKey: "validation.jsonObject",
    },
    [AttributeType.Array]: {
      jsonType: "array",
      errorKey: "validation.jsonArray",
    },
    [AttributeType.Null]: { jsonType: "null", errorKey: "validation.jsonNull" },
  };
  const constraint = constraints[type];
  if (constraint && getJsonType(parsed) !== constraint.jsonType) {
    return { valid: false, errorKey: constraint.errorKey };
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
  | { kind: "match"; Icon: LucideIcon; script: ScriptConfig }
  | { kind: "collect"; collect: InputCollect };

export const getAttributeModifiers = (
  spec: AttributeSpec
): AttributeModifier[] => {
  const modifiers: AttributeModifier[] = [];
  if (spec.optional?.deadline) {
    modifiers.push({ kind: "icon", Icon: IconDuration });
  }
  if (spec.required?.match) {
    modifiers.push({ kind: "match", Icon: IconAttributeMatch, script: spec.required.match });
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
  if (modifier.kind === "collect") return collectTitleKeyMap[modifier.collect];
  if (modifier.kind === "match") return "attribute.modifierMatch";
  if (modifier.Icon === IconDuration) return "attribute.modifierDeadline";
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
