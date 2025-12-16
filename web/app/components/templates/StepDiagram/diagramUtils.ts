import { Step, AttributeRole, ExecutionPlan, FlowContext } from "@/app/api";
import { loadNodePositions } from "./nodePositioning";

export function generateOverviewPlan(
  visibleSteps: Step[]
): ExecutionPlan | null {
  if (visibleSteps.length === 0) return null;

  const attributes: Record<
    string,
    { providers: string[]; consumers: string[] }
  > = {};

  visibleSteps.forEach((step) => {
    Object.entries(step.attributes || {}).forEach(([attrName, attr]) => {
      if (!attributes[attrName]) {
        attributes[attrName] = { providers: [], consumers: [] };
      }

      if (attr.role === AttributeRole.Output) {
        attributes[attrName].providers.push(step.id);
      } else if (
        attr.role === AttributeRole.Required ||
        attr.role === AttributeRole.Optional
      ) {
        attributes[attrName].consumers.push(step.id);
      }
    });
  });

  return {
    attributes,
    steps: Object.fromEntries(visibleSteps.map((s) => [s.id, s])),
    goals: [],
    required: [],
  };
}

export function hasSavedPositions(steps: Step[]): boolean {
  const savedPositions = loadNodePositions();
  return steps.some((step) => savedPositions[step.id]);
}

export function shouldApplyAutoLayout(
  flowData: FlowContext | null,
  visibleSteps: Step[]
): boolean {
  if (flowData) return false; // Only apply in overview mode
  if (visibleSteps.length === 0) return false;
  return !hasSavedPositions(visibleSteps);
}
