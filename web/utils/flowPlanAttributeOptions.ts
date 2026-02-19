import { AttributeRole, ExecutionPlan } from "@/app/api";

export interface FlowInputOption {
  name: string;
  required: boolean;
}

export interface FlowPlanAttributeOptions {
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
}

const isInputRole = (role: AttributeRole): boolean => {
  return role === AttributeRole.Required || role === AttributeRole.Optional;
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
        inputMap.set(name, {
          name,
          required: existing?.required === true || isRequired,
        });
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
