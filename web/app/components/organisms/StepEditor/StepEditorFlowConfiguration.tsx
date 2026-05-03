import React from "react";
import { ExecutionPlan, Step } from "@/app/api";
import { api } from "@/app/api";
import { useT } from "@/app/i18n";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";
import { parseFlowGoals } from "./stepEditorUtils";
import { useFlowFormStepFiltering } from "../FlowCreateForm/useFlowFormStepFiltering";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorFlowConfiguration.module.css";

interface StepEditorFlowConfigurationProps {
  clearPreviewPlan: () => void;
  flowGoals: string;
  flowInitialState: string;
  previewPlan: ExecutionPlan | null;
  setFlowGoals: (value: string) => void;
  setFlowInitialState: (value: string) => void;
  stepId: string;
  steps: Step[];
  updatePreviewPlan: (
    goalSteps: string[],
    initialState: Record<string, any>
  ) => Promise<void>;
}

const StepEditorFlowConfiguration: React.FC<
  StepEditorFlowConfigurationProps
> = ({
  clearPreviewPlan,
  flowGoals,
  flowInitialState,
  previewPlan,
  setFlowGoals,
  setFlowInitialState,
  stepId,
  steps,
  updatePreviewPlan,
}) => {
  const t = useT();
  const goalList = React.useMemo(() => parseFlowGoals(flowGoals), [flowGoals]);
  const sortedSteps = React.useMemo(
    () => [...steps].sort((a, b) => a.name.localeCompare(b.name)),
    [steps]
  );
  const initializedGoalsRef = React.useRef(false);
  const displaySteps = React.useMemo(
    () => sortedSteps.filter((step) => step.id !== stepId),
    [sortedSteps, stepId]
  );

  React.useEffect(() => {
    if (goalList.length === 0) {
      initializedGoalsRef.current = false;
      return;
    }

    if (initializedGoalsRef.current) {
      return;
    }

    initializedGoalsRef.current = true;
    void applyFlowGoalSelectionChange({
      stepIds: goalList,
      initialState: flowInitialState,
      steps: sortedSteps,
      setInitialState: setFlowInitialState,
      setGoalSteps: (ids) => setFlowGoals(ids.join(", ")),
      updatePreviewPlan,
      clearPreviewPlan,
      getExecutionPlan: (goalStepIds, init) =>
        api.getExecutionPlan(goalStepIds, init),
    });
  }, [
    clearPreviewPlan,
    flowInitialState,
    goalList,
    setFlowGoals,
    setFlowInitialState,
    sortedSteps,
    updatePreviewPlan,
  ]);

  React.useEffect(() => {
    if (!goalList.includes(stepId)) {
      return;
    }

    const nextGoals = goalList.filter((id) => id !== stepId);
    void applyFlowGoalSelectionChange({
      stepIds: nextGoals,
      initialState: flowInitialState,
      steps: sortedSteps,
      setInitialState: setFlowInitialState,
      setGoalSteps: (ids) => setFlowGoals(ids.join(", ")),
      updatePreviewPlan,
      clearPreviewPlan,
      getExecutionPlan: (goalStepIds, init) =>
        api.getExecutionPlan(goalStepIds, init),
    });
  }, [
    clearPreviewPlan,
    flowInitialState,
    goalList,
    setFlowGoals,
    setFlowInitialState,
    sortedSteps,
    stepId,
    updatePreviewPlan,
  ]);

  const { included, satisfied, blockedByStep, missingByStep } =
    useFlowFormStepFiltering(displaySteps, flowInitialState, previewPlan);

  const handleGoalToggle = React.useCallback(
    async (goalId: string) => {
      const isSelected = goalList.includes(goalId);
      const nextGoals = isSelected
        ? goalList.filter((id) => id !== goalId)
        : [...goalList, goalId];

      await applyFlowGoalSelectionChange({
        stepIds: nextGoals,
        initialState: flowInitialState,
        steps: sortedSteps,
        setInitialState: setFlowInitialState,
        setGoalSteps: (ids) => setFlowGoals(ids.join(", ")),
        updatePreviewPlan,
        clearPreviewPlan,
        getExecutionPlan: (goalStepIds, init) =>
          api.getExecutionPlan(goalStepIds, init),
      });
    },
    [
      clearPreviewPlan,
      flowInitialState,
      goalList,
      setFlowGoals,
      setFlowInitialState,
      sortedSteps,
      updatePreviewPlan,
    ]
  );

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.flowGoalsLabel")}
        </label>
      </div>
      <div className={localStyles.flowGoalList}>
        {displaySteps.map((step) => {
          const isSelected = goalList.includes(step.id);
          const isIncludedByOthers = included.has(step.id) && !isSelected;
          const isSatisfiedByState = satisfied.has(step.id) && !isSelected;
          const blockedInputs = blockedByStep.get(step.id) || [];
          const isBlocked = blockedInputs.length > 0;
          const missingRequired = missingByStep.get(step.id) || [];
          const isMissing = missingRequired.length > 0;
          const isDisabled =
            isIncludedByOthers || isSatisfiedByState || isBlocked;
          const tooltipText = isIncludedByOthers
            ? t("flowCreate.tooltipAlreadyIncluded")
            : isSatisfiedByState
              ? t("flowCreate.tooltipSatisfiedByState")
              : isBlocked
                ? t("flowCreate.tooltipBlockedByState", {
                    attrs: blockedInputs.join(", "),
                  })
                : isMissing
                  ? t("flowCreate.tooltipMissingRequired", {
                      attrs: missingRequired.join(", "),
                    })
                  : undefined;

          return (
            <button
              key={step.id}
              type="button"
              title={tooltipText}
              onClick={() => {
                if (!isDisabled) {
                  void handleGoalToggle(step.id);
                }
              }}
              disabled={isDisabled}
              className={`${localStyles.flowGoalChip} ${
                isSelected ? localStyles.flowGoalChipSelected : ""
              } ${isIncludedByOthers ? localStyles.flowGoalChipIncluded : ""} ${
                isDisabled ? localStyles.flowGoalChipDisabled : ""
              }`}
            >
              {step.id}
            </button>
          );
        })}
      </div>
    </div>
  );
};

export default StepEditorFlowConfiguration;
