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

export function deriveStepGoalState(
  stepId: string,
  goalIds: string[],
  included: Set<string>,
  satisfied: Set<string>,
  blockedByStep: Map<string, string[]>,
  missingByStep: Map<string, string[]>
): StepGoalState {
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
