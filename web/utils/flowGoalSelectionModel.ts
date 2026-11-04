import { api, ExecutionPlan, GoalSets, Step } from "@/app/api";
import {
  addRequiredDefaults,
  filterDefaultValues,
  parseState,
} from "@/utils/stateUtils";
import { generatePadded } from "@/utils/flowUtils";
import { nonEmptyGoalSets } from "@/utils/goalSets";
import { formatJSON } from "@/utils/jsonUtils";

export interface ApplyFlowGoalSelectionChangeParams {
  goals: GoalSets;
  initialState: string;
  steps: Step[];
  idManuallyEdited?: boolean;
  setNewID?: (id: string) => void;
  setInitialState: (state: string) => void;
  setGoalSteps: (goals: GoalSets) => void;
  updatePreviewPlan: (
    goalSteps: GoalSets,
    initialState: Record<string, any>,
    spaceId?: string
  ) => Promise<void>;
  setPreviewPlan?: (plan: ExecutionPlan | null) => void;
  clearPreviewPlan: () => void;
  spaceId?: string;
}

export async function applyFlowGoalSelectionChange({
  goals,
  initialState,
  steps,
  idManuallyEdited,
  setNewID,
  setInitialState,
  setGoalSteps,
  updatePreviewPlan,
  setPreviewPlan,
  clearPreviewPlan,
  spaceId,
}: ApplyFlowGoalSelectionChangeParams): Promise<void> {
  const currentState = parseState(initialState);
  const nonDefaultState = filterDefaultValues(currentState, steps);
  const planGoals = nonEmptyGoalSets(goals);

  if (planGoals.length === 0) {
    setInitialState(formatJSON(nonDefaultState));
    setPreviewPlan?.(null);
    clearPreviewPlan();
    setGoalSteps(goals);
    return;
  }

  try {
    const executionPlan = await api.getExecutionPlan({
      goalSteps: planGoals,
      initialState: nonDefaultState,
      spaceId,
    });
    setPreviewPlan?.(executionPlan);

    const stateWithDefaults = addRequiredDefaults(
      nonDefaultState,
      executionPlan
    );

    setInitialState(formatJSON(stateWithDefaults));

    if (!idManuallyEdited && setNewID) {
      const lastGoalId = planGoals.flat().at(-1)!;
      const goalStep = steps.find((s) => s.id === lastGoalId);
      const goalName = goalStep?.name || lastGoalId;
      const kebabName = goalName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
      setNewID(`${kebabName}-${generatePadded()}`);
    }

    setGoalSteps(goals);
    await updatePreviewPlan(planGoals, nonDefaultState, spaceId);
  } catch {
    setPreviewPlan?.(null);
    clearPreviewPlan();
    setGoalSteps(goals);
  }
}
