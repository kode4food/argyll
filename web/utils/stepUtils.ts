import {
  Step,
  AttributeRole,
  AttributeSpec,
  InputCollect,
  ScriptConfig,
} from "@/app/api";
import { stepTypeOrder } from "@/app/constants";
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

export const roleArgType: Record<AttributeRole, ArgType> = {
  required: "required",
  optional: "optional",
  const: "const",
  meta: "meta",
  feed: "feed",
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
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Feed),
    ...sortedByName.filter((a) => a.spec.role === AttributeRole.Output),
  ];
};

const SPEC_PREVIEWS: Partial<
  Record<AttributeRole, (spec: AttributeSpec) => string | undefined>
> = {
  [AttributeRole.Optional]: (spec) => spec.optional?.default,
  [AttributeRole.Const]: (spec) => spec.const?.value,
  [AttributeRole.Meta]: (spec) => spec.meta?.key,
  [AttributeRole.Feed]: (spec) => spec.feed?.id,
};

// The value a Step declares for an Attribute: default, const, meta key, Feed
export const getSpecPreview = (spec: AttributeSpec): string | undefined =>
  SPEC_PREVIEWS[spec.role]?.(spec) || undefined;

type ValidationResult = {
  valid: boolean;
  errorKey?: string;
  errorVars?: Record<string, string>;
};

export const validateDefaultValue = (value: string): ValidationResult => {
  if (!value.trim()) {
    return { valid: true };
  }
  try {
    JSON.parse(value.trim());
  } catch {
    return { valid: false, errorKey: "validation.jsonInvalid" };
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
  const match = spec.required?.match ?? spec.optional?.match;
  if (match) {
    modifiers.push({ kind: "match", Icon: IconAttributeMatch, script: match });
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
    const orderDiff = stepTypeOrder[aType] - stepTypeOrder[bType];
    if (orderDiff !== 0) return orderDiff;
    return a.name.localeCompare(b.name);
  });
};
