import { ExecutionPlan, Step } from "@/app/api";

export function getStepsFromPlan(
  plan: ExecutionPlan | undefined | null
): Step[] {
  if (!plan?.steps) {
    return [];
  }
  return Object.values(plan.steps);
}
