import { Step, AttributeRole, ExecutionPlan } from "@/app/api";
import { loadNodePositions, NodePositionScope } from "@/utils/nodePositioning";

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

export function hasSavedPositions(
  steps: Step[],
  scope?: NodePositionScope
): boolean {
  const savedPositions = loadNodePositions(scope);
  return steps.some((step) => savedPositions[step.id]);
}

export function shouldApplyAutoLayout(
  visibleSteps: Step[],
  scope?: NodePositionScope
): boolean {
  if (visibleSteps.length === 0) return false;
  return !hasSavedPositions(visibleSteps, scope);
}
