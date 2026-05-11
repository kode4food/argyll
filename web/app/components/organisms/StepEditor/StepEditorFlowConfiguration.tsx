import React from "react";
import { ExecutionPlan, Step } from "@/app/api";
import { api } from "@/app/api";
import { useT } from "@/app/i18n";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";
import {
  deriveStepGoalState,
  getGoalTooltip,
  StepGoalState,
} from "@/utils/flowGoalStepState";
import { parseFlowGoals } from "./stepEditorUtils";
import { useFlowFormStepFiltering } from "../FlowCreateForm/useFlowFormStepFiltering";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorFlowConfiguration.module.css";

type TFn = (key: string, vars?: Record<string, string | number>) => string;

interface GoalChipProps {
  step: Step;
  state: StepGoalState;
  onToggle: (id: string) => void;
  t: TFn;
}

const GoalChip: React.FC<GoalChipProps> = ({ step, state, onToggle, t }) => (
  <button
    type="button"
    title={getGoalTooltip(state, t)}
    onClick={() => {
      if (!state.isDisabled) onToggle(step.id);
    }}
    disabled={state.isDisabled}
    className={`${localStyles.flowGoalChip} ${
      state.isSelected ? localStyles.flowGoalChipSelected : ""
    } ${state.isIncludedByOthers ? localStyles.flowGoalChipIncluded : ""} ${
      state.isDisabled ? localStyles.flowGoalChipDisabled : ""
    }`}
  >
    {step.id}
  </button>
);

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
        {displaySteps.map((step) => (
          <GoalChip
            key={step.id}
            step={step}
            state={deriveStepGoalState(step.id, goalList, {
              included,
              satisfied,
              blockedByStep,
              missingByStep,
            })}
            onToggle={handleGoalToggle}
            t={t}
          />
        ))}
      </div>
    </div>
  );
};

export default StepEditorFlowConfiguration;
