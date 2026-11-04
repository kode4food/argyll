import { DEFAULT_HTTP_METHOD, Step, toGoalSets } from "@/app/api";
import { formatGoals } from "@/utils/goalSets";
import { getStepActionIcon } from "@/utils/iconRegistry";
import { formatScriptPreview } from "@/utils/stepFooterUtils";
import { type DisplayInfo } from "./StepInfoDisplay";

export const flowGoalsText = (step: Step): string | undefined =>
  step.type === "flow" && step.flow?.goals?.steps.length
    ? formatGoals(toGoalSets(step.flow.goals))
    : undefined;

export const stepDisplayInfo = (step: Step): DisplayInfo => {
  const icon = getStepActionIcon(step);
  const goals = flowGoalsText(step);

  if (step.type === "script" && step.script) {
    return {
      icon,
      text: formatScriptPreview(step.script.script, step.script.language),
      className: "endpoint-script",
    };
  }
  if (goals) return { icon, text: goals };
  if (step.http) {
    const method = step.http.invoke?.method || DEFAULT_HTTP_METHOD;
    return { icon, text: `${method} ${step.http.invoke?.endpoint}` };
  }
  return null;
};
