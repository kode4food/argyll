import { Step, AttributeRole, AttributeSpec, AttributeType } from "@/app/api";

export type StepType = "resolver" | "processor" | "collector" | "neutral";

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
): { valid: boolean; error?: string } => {
  if (!value.trim()) {
    return { valid: true };
  }

  const trimmed = value.trim();

  let parsed: any;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { valid: false, error: "Must be valid JSON" };
  }

  if (type === AttributeType.Any) {
    return { valid: true };
  }

  const jsonType = getJsonType(parsed);

  switch (type) {
    case AttributeType.String:
      if (jsonType !== "string") {
        return { valid: false, error: "Must be a valid JSON string" };
      }
      break;

    case AttributeType.Number:
      if (jsonType !== "number") {
        return { valid: false, error: "Must be a valid number" };
      }
      break;

    case AttributeType.Boolean:
      if (jsonType !== "boolean") {
        return { valid: false, error: 'Must be "true" or "false"' };
      }
      break;

    case AttributeType.Object:
      if (jsonType !== "object") {
        return { valid: false, error: "Must be a valid JSON object" };
      }
      break;

    case AttributeType.Array:
      if (jsonType !== "array") {
        return { valid: false, error: "Must be a valid JSON array" };
      }
      break;

    case AttributeType.Null:
      if (jsonType !== "null") {
        return { valid: false, error: 'Must be "null"' };
      }
      break;
  }

  return { valid: true };
};

export const getStepType = (step: Step): StepType => {
  if (!step.attributes) return "neutral";

  const hasRequiredInputs = Object.values(step.attributes).some(
    (attr) => attr.role === AttributeRole.Required
  );
  const hasOutputs = Object.values(step.attributes).some(
    (attr) => attr.role === AttributeRole.Output
  );

  if (hasOutputs && !hasRequiredInputs) return "resolver";
  if (!hasOutputs && hasRequiredInputs) return "collector";
  if (hasOutputs && hasRequiredInputs) return "processor";
  return "neutral";
};

export const getStepTypeLabel = (stepType: StepType): string => {
  switch (stepType) {
    case "resolver":
      return "R";
    case "collector":
      return "C";
    case "processor":
      return "P";
    case "neutral":
      return "S";
  }
};

export const sortStepsByType = (steps: Step[]): Step[] => {
  const stepOrder = { collector: 1, processor: 2, resolver: 3, neutral: 4 };
  return [...steps].sort((a, b) => {
    const aType = getStepType(a);
    const bType = getStepType(b);
    const orderDiff = stepOrder[aType] - stepOrder[bType];
    if (orderDiff !== 0) return orderDiff;
    return a.name.localeCompare(b.name);
  });
};
