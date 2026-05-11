import { AttributeRole, AttributeType, ExecutionPlan, Step } from "@/app/api";
import {
  FlowInputMeta,
  getInputPriorityRank,
  getOrCreateFlowInputMeta,
  getTypeDefaultValue,
  mergeExplicitDefault,
  mergeInputType,
  normalizeDefaultValue,
} from "./flowPlanInputHelpers";

export interface FlowInputOption {
  name: string;
  required: boolean;
  type?: AttributeType;
  defaultValue?: string;
  unreachable?: boolean;
  satisfiedByOutput?: boolean;
}

export interface FlowPlanAttributeOptions {
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
}

interface AttributeCollectionContext {
  requiredInputs: Set<string>;
  inputMap: Map<string, FlowInputOption>;
  inputMetaMap: Map<string, FlowInputMeta>;
}

interface FlowAttributeSets {
  inputMap: Map<string, FlowInputOption>;
  outputSet: Set<string>;
}

const isInputRole = (role: AttributeRole): boolean => {
  return role === AttributeRole.Required || role === AttributeRole.Optional;
};

const processInputAttribute = (
  name: string,
  spec: {
    role: AttributeRole;
    type?: AttributeType;
    optional?: { default?: string };
  },
  ctx: AttributeCollectionContext
): void => {
  const { requiredInputs, inputMap, inputMetaMap } = ctx;
  const existing = inputMap.get(name);
  const meta = getOrCreateFlowInputMeta(inputMetaMap, name);
  const isRequired = requiredInputs.has(name);
  const mergedType = mergeInputType(existing?.type, spec.type);
  meta.hasRequiredSpec =
    meta.hasRequiredSpec || spec.role === AttributeRole.Required;
  if (spec.optional?.default !== undefined) {
    meta.hasSpecDefault = true;
    const normalizedSpecDefault =
      normalizeDefaultValue(spec.optional.default) ?? "";
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
};

export const getFlowPlanAttributeOptions = (
  plan: ExecutionPlan | null,
  catalogSteps?: Step[]
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
        processInputAttribute(name, spec, {
          requiredInputs,
          inputMap,
          inputMetaMap,
        });
        return;
      }
      if (spec.role === AttributeRole.Output) {
        outputSet.add(name);
      }
    });
  });

  const unreachableInputs = collectUnreachableInputs(plan, catalogSteps, {
    inputMap,
    outputSet,
  });

  return {
    flowInputOptions: Array.from(inputMap.values())
      .map((option) => {
        if (outputSet.has(option.name)) {
          return { ...option, satisfiedByOutput: true };
        }
        const meta = inputMetaMap.get(option.name);
        if (meta?.hasRequiredSpec && !option.required) {
          return { ...option, required: true };
        }
        return option;
      })
      .concat(unreachableInputs)
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

const collectUnreachableInputs = (
  plan: ExecutionPlan,
  catalogSteps: Step[] | undefined,
  sets: FlowAttributeSets
): FlowInputOption[] => {
  const { inputMap, outputSet } = sets;
  const missingMap = plan.excluded?.missing;
  if (!missingMap || !catalogSteps) {
    return [];
  }

  const catalogByID = new Map(catalogSteps.map((s) => [s.id, s]));
  const seen = new Set<string>();
  const result: FlowInputOption[] = [];

  for (const [stepID, missingNames] of Object.entries(missingMap)) {
    const step = catalogByID.get(stepID);
    if (!step) {
      continue;
    }
    for (const name of missingNames) {
      if (inputMap.has(name) || outputSet.has(name) || seen.has(name)) {
        continue;
      }
      seen.add(name);
      const spec = step.attributes[name];
      const option: FlowInputOption = {
        name,
        required: false,
        unreachable: true,
      };
      if (spec?.type) {
        option.type = spec.type;
      }
      result.push(option);
    }
  }

  return result;
};
