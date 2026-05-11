export interface StepGoalState {
  isSelected: boolean;
  isIncludedByOthers: boolean;
  isSatisfiedByState: boolean;
  blockedInputs: string[];
  isBlocked: boolean;
  missingRequired: string[];
  isMissing: boolean;
  isDisabled: boolean;
}

export interface GoalStepContext {
  included: Set<string>;
  satisfied: Set<string>;
  blockedByStep: Map<string, string[]>;
  missingByStep: Map<string, string[]>;
}

type TFn = (key: string, vars?: Record<string, string | number>) => string;

export function getGoalTooltip(
  state: StepGoalState,
  t: TFn
): string | undefined {
  if (state.isIncludedByOthers) return t("flowCreate.tooltipAlreadyIncluded");
  if (state.isSatisfiedByState) return t("flowCreate.tooltipSatisfiedByState");
  if (state.blockedInputs.length > 0)
    return t("flowCreate.tooltipBlockedByState", {
      attrs: state.blockedInputs.join(", "),
    });
  if (state.isMissing)
    return t("flowCreate.tooltipMissingRequired", {
      attrs: state.missingRequired.join(", "),
    });
  return undefined;
}

export function deriveStepGoalState(
  stepId: string,
  goalIds: string[],
  context: GoalStepContext
): StepGoalState {
  const { included, satisfied, blockedByStep, missingByStep } = context;
  const isSelected = goalIds.includes(stepId);
  const isIncludedByOthers = included.has(stepId) && !isSelected;
  const isSatisfiedByState = satisfied.has(stepId) && !isSelected;
  const blockedInputs = blockedByStep.get(stepId) ?? [];
  const isBlocked = blockedInputs.length > 0;
  const missingRequired = missingByStep.get(stepId) ?? [];
  const isMissing = missingRequired.length > 0;
  const isDisabled = isIncludedByOthers || isSatisfiedByState || isBlocked;
  return {
    isSelected,
    isIncludedByOthers,
    isSatisfiedByState,
    blockedInputs,
    isBlocked,
    missingRequired,
    isMissing,
    isDisabled,
  };
}
