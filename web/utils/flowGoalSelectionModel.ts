import { ExecutionPlan, Step } from "@/app/api";
import {
  addRequiredDefaults,
  filterDefaultValues,
  parseState,
} from "@/utils/stateUtils";

const JSON_INDENT_SPACES = 2;

export type GetExecutionPlan = (
  stepIds: string[],
  initialState: Record<string, any>
) => Promise<ExecutionPlan>;

export interface ApplyFlowGoalSelectionChangeParams {
  stepIds: string[];
  initialState: string;
  steps: Step[];
  idManuallyEdited?: boolean;
  setNewID?: (id: string) => void;
  generatePadded?: () => string;
  setInitialState: (state: string) => void;
  setGoalSteps: (stepIds: string[]) => void;
  updatePreviewPlan: (
    goalSteps: string[],
    initialState: Record<string, any>
  ) => Promise<void>;
  setPreviewPlan?: (plan: ExecutionPlan | null) => void;
  clearPreviewPlan: () => void;
  getExecutionPlan: GetExecutionPlan;
}

export async function applyFlowGoalSelectionChange({
  stepIds,
  initialState,
  steps,
  idManuallyEdited,
  setNewID,
  generatePadded,
  setInitialState,
  setGoalSteps,
  updatePreviewPlan,
  setPreviewPlan,
  clearPreviewPlan,
  getExecutionPlan,
}: ApplyFlowGoalSelectionChangeParams): Promise<void> {
  const currentState = parseState(initialState);
  const nonDefaultState = filterDefaultValues(currentState, steps);

  if (stepIds.length === 0) {
    setInitialState(JSON.stringify(nonDefaultState, null, JSON_INDENT_SPACES));
    setPreviewPlan?.(null);
    clearPreviewPlan();
    setGoalSteps([]);
    return;
  }

  try {
    const executionPlan = await getExecutionPlan(stepIds, nonDefaultState);
    setPreviewPlan?.(executionPlan);

    const stateWithDefaults = addRequiredDefaults(
      nonDefaultState,
      executionPlan
    );

    setInitialState(
      JSON.stringify(stateWithDefaults, null, JSON_INDENT_SPACES)
    );

    if (!idManuallyEdited && setNewID && generatePadded) {
      const lastGoalId = stepIds[stepIds.length - 1];
      const goalStep = steps.find((s) => s.id === lastGoalId);
      const goalName = goalStep?.name || lastGoalId;
      const kebabName = goalName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
      setNewID(`${kebabName}-${generatePadded()}`);
    }

    if (stepIds.length > 1) {
      const lastGoal = stepIds[stepIds.length - 1];
      const previousGoals = stepIds.slice(0, -1);

      try {
        const lastGoalPlan = await getExecutionPlan([lastGoal], {});
        const lastGoalStepIds = new Set(Object.keys(lastGoalPlan.steps || {}));

        const remainingGoals = previousGoals.filter(
          (id) => !lastGoalStepIds.has(id)
        );

        const finalGoals = [...remainingGoals, lastGoal];

        if (finalGoals.length !== stepIds.length) {
          setGoalSteps(finalGoals);
          await updatePreviewPlan(finalGoals, nonDefaultState);
          return;
        }
      } catch {}
    }

    setGoalSteps(stepIds);
    await updatePreviewPlan(stepIds, nonDefaultState);
  } catch {
    setPreviewPlan?.(null);
    clearPreviewPlan();
    setGoalSteps(stepIds);
  }
}
