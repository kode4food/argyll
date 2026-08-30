import React from "react";
import { ExecutionPlan, Step } from "@/app/api";
import { useT } from "@/app/i18n";
import { IconCompensate, IconFlowGoals, IconSpace } from "@/utils/iconRegistry";
import IconCheckbox from "@/app/components/molecules/IconCheckbox";
import { getStepActionIcon } from "@/utils/iconRegistry";
import { getStepType } from "@/utils/stepUtils";
import SelectField from "@/app/components/molecules/SelectField";
import { useSpaces, useSpaceSelection } from "@/app/store/flowStore";
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

const GoalChip: React.FC<GoalChipProps> = ({ step, state, onToggle, t }) => {
  const TypeIcon = getStepActionIcon(step);

  return (
    <button
      type="button"
      aria-label={step.id}
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
      <TypeIcon className={`step-type-icon ${getStepType(step)}`} />
      {step.id}
    </button>
  );
};

interface StepEditorFlowConfigurationProps {
  clearPreviewPlan: () => void;
  flowCompensate: boolean;
  flowGoals: string;
  flowInitialState: string;
  flowSpaceId: string;
  previewPlan: ExecutionPlan | null;
  setFlowCompensate: (value: boolean) => void;
  setFlowGoals: (value: string) => void;
  setFlowInitialState: (value: string) => void;
  setFlowSpaceId: (value: string) => void;
  stepId: string;
  steps: Step[];
  updatePreviewPlan: (
    goalSteps: string[],
    initialState: Record<string, any>,
    spaceId?: string
  ) => Promise<void>;
}

const StepEditorFlowConfiguration: React.FC<
  StepEditorFlowConfigurationProps
> = ({
  clearPreviewPlan,
  flowCompensate,
  flowGoals,
  flowInitialState,
  flowSpaceId,
  previewPlan,
  setFlowCompensate,
  setFlowGoals,
  setFlowInitialState,
  setFlowSpaceId,
  stepId,
  steps,
  updatePreviewPlan,
}) => {
  const t = useT();
  const spaces = useSpaces();
  const spaceSelection = useSpaceSelection();
  const goalList = React.useMemo(() => parseFlowGoals(flowGoals), [flowGoals]);
  const sortedSteps = React.useMemo(
    () => [...steps].sort((a, b) => a.name.localeCompare(b.name)),
    [steps]
  );
  const initializedGoalsRef = React.useRef(false);
  const spaceOptions = React.useMemo(
    () => [
      { value: "", label: t("flowCreate.spaceAll") },
      ...spaces.map((s) => ({
        value: s.id,
        label: s.name,
        title: s.description,
      })),
    ],
    [spaces, t]
  );
  // The engine plans within the Space, so a goal outside it could never run
  const displaySteps = React.useMemo(() => {
    const selected = flowSpaceId ? spaceSelection[flowSpaceId] : null;
    return sortedSteps.filter(
      (step) => step.id !== stepId && (!selected || selected.has(step.id))
    );
  }, [flowSpaceId, sortedSteps, spaceSelection, stepId]);

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
      spaceId: flowSpaceId || undefined,
    });
  }, [
    clearPreviewPlan,
    flowInitialState,
    goalList,
    setFlowGoals,
    setFlowInitialState,
    sortedSteps,
    flowSpaceId,
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
      spaceId: flowSpaceId || undefined,
    });
  }, [
    clearPreviewPlan,
    flowInitialState,
    goalList,
    setFlowGoals,
    setFlowInitialState,
    sortedSteps,
    stepId,
    flowSpaceId,
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
        spaceId: flowSpaceId || undefined,
      });
    },
    [
      clearPreviewPlan,
      flowInitialState,
      goalList,
      setFlowGoals,
      setFlowInitialState,
      sortedSteps,
      flowSpaceId,
      updatePreviewPlan,
    ]
  );

  const handleSpaceChange = React.useCallback(
    (value: string) => {
      setFlowSpaceId(value);

      const selected = value ? spaceSelection[value] : null;
      const nextGoals = selected
        ? goalList.filter((id) => selected.has(id))
        : goalList;

      setFlowGoals(nextGoals.join(", "));
      clearPreviewPlan();
      void applyFlowGoalSelectionChange({
        stepIds: nextGoals,
        initialState: flowInitialState,
        steps: sortedSteps,
        setInitialState: setFlowInitialState,
        setGoalSteps: (ids) => setFlowGoals(ids.join(", ")),
        updatePreviewPlan,
        clearPreviewPlan,
        spaceId: value || undefined,
      });
    },
    [
      clearPreviewPlan,
      flowInitialState,
      goalList,
      setFlowGoals,
      setFlowInitialState,
      setFlowSpaceId,
      sortedSteps,
      spaceSelection,
      updatePreviewPlan,
    ]
  );

  return (
    <div className={formStyles.section}>
      <div className={localStyles.flowGoalRow}>
        <div className={localStyles.flowSpaceColumn}>
          <label className={localStyles.flowSpaceLabel}>
            <span className={formStyles.labelIcon}>
              <IconSpace aria-hidden="true" />
            </span>
            {t("flowCreate.spaceLabel")}
          </label>
          <SelectField
            ariaLabel={t("flowCreate.spaceLabel")}
            onChange={handleSpaceChange}
            options={spaceOptions}
            value={flowSpaceId}
          />
        </div>
        <div className={localStyles.flowGoalColumn}>
          <div className={localStyles.flowLabelRow}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconFlowGoals aria-hidden="true" />
              </span>
              {t("stepEditor.flowGoalsLabel")}
            </label>
            <IconCheckbox
              checked={flowCompensate}
              Icon={IconCompensate}
              label={t("stepEditor.flowCompensateLabel")}
              onChange={setFlowCompensate}
              title={t("stepEditor.flowCompensateTitle")}
            />
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
      </div>
    </div>
  );
};

export default StepEditorFlowConfiguration;
