import { AttributeRole, AttributeType, ExecutionPlan } from "@/app/api";

export interface FlowInputOption {
  name: string;
  required: boolean;
  type?: AttributeType;
  defaultValue?: string;
}

export interface FlowPlanAttributeOptions {
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
}

const isInputRole = (role: AttributeRole): boolean => {
  return role === AttributeRole.Required || role === AttributeRole.Optional;
};

const mergeInputType = (
  existingType: AttributeType | undefined,
  nextType: AttributeType | undefined
): AttributeType | undefined => {
  if (!existingType) {
    return nextType;
  }
  if (!nextType || existingType === nextType) {
    return existingType;
  }
  return AttributeType.Any;
};

const getTypeDefaultValue = (attrType?: AttributeType): string | undefined => {
  switch (attrType) {
    case AttributeType.Number:
      return "0";
    case AttributeType.Boolean:
      return "false";
    case AttributeType.Object:
      return "{}";
    case AttributeType.Array:
      return "[]";
    case AttributeType.Null:
      return "null";
    default:
      return undefined;
  }
};

const normalizeDefaultValue = (defaultValue?: string): string | undefined => {
  if (defaultValue === undefined) {
    return undefined;
  }
  const trimmed = defaultValue.trim();
  if (!trimmed) {
    return "";
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (parsed === null || parsed === undefined) {
      return "";
    }
    if (typeof parsed === "string") {
      return parsed;
    }
    return JSON.stringify(parsed);
  } catch {
    return trimmed;
  }
};

const collectFlowAttributeOptions = (
  plan: ExecutionPlan | null
): FlowPlanAttributeOptions => {
  const steps = plan?.steps;
  if (!steps) {
    return { flowInputOptions: [], flowOutputOptions: [] };
  }

  const requiredInputs = new Set(plan?.required || []);
  const inputMap = new Map<string, FlowInputOption>();
  const outputSet = new Set<string>();

  Object.values(steps).forEach((planStep) => {
    Object.entries(planStep.attributes || {}).forEach(([name, spec]) => {
      if (isInputRole(spec.role)) {
        const existing = inputMap.get(name);
        const isRequired = requiredInputs.has(name);
        const mergedType = mergeInputType(existing?.type, spec.type);
        const defaultValue =
          normalizeDefaultValue(spec.default) ??
          getTypeDefaultValue(mergedType);
        const mergedDefaultValue =
          existing?.defaultValue === undefined
            ? defaultValue
            : defaultValue === undefined ||
                existing.defaultValue === defaultValue
              ? existing.defaultValue
              : undefined;
        const nextInput: FlowInputOption = {
          name,
          required: existing?.required === true || isRequired,
          type: mergedType,
        };
        if (mergedDefaultValue !== undefined) {
          nextInput.defaultValue = mergedDefaultValue;
        }
        inputMap.set(name, nextInput);
        return;
      }

      if (spec.role === AttributeRole.Output) {
        outputSet.add(name);
      }
    });
  });

  return {
    flowInputOptions: Array.from(inputMap.values()).sort((a, b) =>
      a.name.localeCompare(b.name)
    ),
    flowOutputOptions: Array.from(outputSet).sort((a, b) => a.localeCompare(b)),
  };
};

export const getFlowPlanAttributeOptions = (
  plan: ExecutionPlan | null
): FlowPlanAttributeOptions => {
  return collectFlowAttributeOptions(plan);
};
