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

interface FlowInputMeta {
  hasRequiredSpec: boolean;
  hasSpecDefault: boolean;
  explicitDefault?: string;
  hasConflictingDefaults: boolean;
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

const getOrCreateFlowInputMeta = (
  inputMetaMap: Map<string, FlowInputMeta>,
  name: string
): FlowInputMeta => {
  const existing = inputMetaMap.get(name);
  if (existing) {
    return existing;
  }

  const created: FlowInputMeta = {
    hasRequiredSpec: false,
    hasSpecDefault: false,
    hasConflictingDefaults: false,
  };
  inputMetaMap.set(name, created);
  return created;
};

const mergeExplicitDefault = (
  meta: FlowInputMeta,
  normalizedDefault: string
): void => {
  if (meta.hasConflictingDefaults) {
    return;
  }
  if (meta.explicitDefault === undefined) {
    meta.explicitDefault = normalizedDefault;
    return;
  }
  if (meta.explicitDefault !== normalizedDefault) {
    meta.hasConflictingDefaults = true;
    meta.explicitDefault = undefined;
  }
};

export const getFlowPlanAttributeOptions = (
  plan: ExecutionPlan | null
): FlowPlanAttributeOptions => {
  const steps = plan?.steps;
  if (!steps) {
    return { flowInputOptions: [], flowOutputOptions: [] };
  }

  const requiredInputs = new Set(plan?.required || []);
  const inputMap = new Map<string, FlowInputOption>();
  const inputMetaMap = new Map<string, FlowInputMeta>();
  const outputSet = new Set<string>();

  Object.values(steps).forEach((planStep) => {
    Object.entries(planStep.attributes || {}).forEach(([name, spec]) => {
      if (isInputRole(spec.role)) {
        const existing = inputMap.get(name);
        const meta = getOrCreateFlowInputMeta(inputMetaMap, name);
        const isRequired = requiredInputs.has(name);
        const mergedType = mergeInputType(existing?.type, spec.type);
        meta.hasRequiredSpec =
          meta.hasRequiredSpec || spec.role === AttributeRole.Required;
        if (spec.default !== undefined) {
          meta.hasSpecDefault = true;
          const normalizedSpecDefault =
            normalizeDefaultValue(spec.default) ?? "";
          mergeExplicitDefault(meta, normalizedSpecDefault);
        }
        const fallbackTypeDefault = getTypeDefaultValue(mergedType);
        const resolvedDefault = meta.hasConflictingDefaults
          ? fallbackTypeDefault
          : (meta.explicitDefault ?? fallbackTypeDefault);
        const nextInput: FlowInputOption = {
          name,
          required: existing?.required === true || isRequired,
          type: mergedType,
        };
        if (resolvedDefault !== undefined) {
          nextInput.defaultValue = resolvedDefault;
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
    flowInputOptions: Array.from(inputMap.values())
      .map((option) => {
        const meta = inputMetaMap.get(option.name);
        if (
          meta?.hasRequiredSpec &&
          !outputSet.has(option.name) &&
          !option.required
        ) {
          return {
            ...option,
            required: true,
          };
        }
        return option;
      })
      .sort((a, b) => {
        const rankA = getInputPriorityRank(a, outputSet, inputMetaMap);
        const rankB = getInputPriorityRank(b, outputSet, inputMetaMap);
        if (rankA !== rankB) {
          return rankA - rankB;
        }
        return a.name.localeCompare(b.name);
      }),
    flowOutputOptions: Array.from(outputSet).sort((a, b) => a.localeCompare(b)),
  };
};

const getInputPriorityRank = (
  option: FlowInputOption,
  outputSet: Set<string>,
  inputMetaMap: Map<string, FlowInputMeta>
): number => {
  const meta = inputMetaMap.get(option.name);
  if (outputSet.has(option.name)) {
    return 3;
  }
  if (option.required || meta?.hasRequiredSpec) {
    return 0;
  }
  if (meta?.hasSpecDefault) {
    return 2;
  }
  return 1;
};
