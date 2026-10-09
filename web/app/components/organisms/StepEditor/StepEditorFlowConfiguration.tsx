import React from "react";
import { ExecutionPlan, GoalSets, Step } from "@/app/api";
import { useT } from "@/app/i18n";
import {
  IconClose,
  IconCompensate,
  IconFlowGoals,
  IconGoalFallback,
  IconSpace,
} from "@/utils/iconRegistry";
import IconCheckbox from "@/app/components/molecules/IconCheckbox";
import { getStepActionIcon } from "@/utils/iconRegistry";
import { getStepType } from "@/utils/stepUtils";
import SelectField from "@/app/components/molecules/SelectField";
import ComboInput from "@/app/components/molecules/ComboInput";
import { useSpaces, useSpaceSelection } from "@/app/store/flowStore";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";
import {
  deriveStepGoalState,
  GoalStepContext,
} from "@/utils/flowGoalStepState";
import { addGoal, upstreamSteps } from "@/utils/goalSets";
import { useFlowFormStepFiltering } from "../FlowCreateForm/useFlowFormStepFiltering";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorFlowConfiguration.module.css";

type TFn = (key: string, vars?: Record<string, string | number>) => string;

interface GoalSetLineProps {
  context: Omit<GoalStepContext, "included">;
  onChange: (set: string[]) => void;
  set: string[];
  steps: Step[];
  t: TFn;
}

// GoalSetLine edits one AND-set. Its suggestions exclude only the Steps
// upstream of goals on this line, never those of other OR lines
const GoalSetLine: React.FC<GoalSetLineProps> = ({
  context,
  onChange,
  set,
  steps,
  t,
}) => {
  const [draft, setDraft] = React.useState("");
  const suggestions = React.useMemo(() => {
    const included = upstreamSteps(steps, set);
    return steps.filter(
      (step) =>
        !set.includes(step.id) &&
        !deriveStepGoalState(step.id, set, { ...context, included }).isDisabled
    );
  }, [context, set, steps]);

  const handleDraftChange = (value: string) => {
    if (suggestions.some((step) => step.id === value)) {
      setDraft("");
      onChange(addGoal(steps, set, value));
      return;
    }
    setDraft(value);
  };

  return (
    <div className={localStyles.flowGoalList} data-testid="flow-goal-line">
      {set.map((id) => {
        const step = steps.find((s) => s.id === id);
        const TypeIcon = step ? getStepActionIcon(step) : null;
        return (
          <span
            key={id}
            className={`${localStyles.flowGoalChip} ${localStyles.flowGoalChipSelected}`}
          >
            {TypeIcon && step && (
              <TypeIcon className={`step-type-icon ${getStepType(step)}`} />
            )}
            {id}
            <button
              type="button"
              className={localStyles.flowGoalRemove}
              aria-label={t("goals.removeGoal", { id })}
              onClick={() => onChange(set.filter((goal) => goal !== id))}
            >
              <IconClose aria-hidden="true" />
            </button>
          </span>
        );
      })}
      <ComboInput
        ariaLabel={t("goals.addGoal")}
        className={localStyles.flowGoalInput}
        onChange={handleDraftChange}
        placeholder={t("goals.addGoal")}
        suggestions={suggestions.map((step) => step.id)}
        value={draft}
      />
    </div>
  );
};

interface StepEditorFlowConfigurationProps {
  clearPreviewPlan: () => void;
  flowCompensate: boolean;
  flowGoals: GoalSets;
  flowInitialState: string;
  flowSpaceId: string;
  previewPlan: ExecutionPlan | null;
  setFlowCompensate: (value: boolean) => void;
  setFlowGoals: (value: GoalSets) => void;
  setFlowInitialState: (value: string) => void;
  setFlowSpaceId: (value: string) => void;
  stepId: string;
  steps: Step[];
  updatePreviewPlan: (
    goalSteps: GoalSets,
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
  const lines = React.useMemo<GoalSets>(
    () => (flowGoals.length > 0 ? flowGoals : [[]]),
    [flowGoals]
  );
  const hasGoals = flowGoals.some((set) => set.length > 0);
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

  const applyGoals = React.useCallback(
    (goals: GoalSets, spaceId: string) =>
      applyFlowGoalSelectionChange({
        goals,
        initialState: flowInitialState,
        steps: sortedSteps,
        setInitialState: setFlowInitialState,
        setGoalSteps: setFlowGoals,
        updatePreviewPlan,
        clearPreviewPlan,
        spaceId: spaceId || undefined,
      }),
    [
      clearPreviewPlan,
      flowInitialState,
      setFlowGoals,
      setFlowInitialState,
      sortedSteps,
      updatePreviewPlan,
    ]
  );

  React.useEffect(() => {
    if (!hasGoals) {
      initializedGoalsRef.current = false;
      return;
    }

    if (initializedGoalsRef.current) {
      return;
    }

    initializedGoalsRef.current = true;
    void applyGoals(flowGoals, flowSpaceId);
  }, [applyGoals, flowGoals, flowSpaceId, hasGoals]);

  React.useEffect(() => {
    if (!flowGoals.some((set) => set.includes(stepId))) {
      return;
    }

    void applyGoals(
      flowGoals.map((set) => set.filter((id) => id !== stepId)),
      flowSpaceId
    );
  }, [applyGoals, flowGoals, flowSpaceId, stepId]);

  const { satisfied, blockedByStep, missingByStep } = useFlowFormStepFiltering(
    displaySteps,
    flowInitialState,
    previewPlan
  );
  const context = React.useMemo(
    () => ({ satisfied, blockedByStep, missingByStep }),
    [satisfied, blockedByStep, missingByStep]
  );

  const handleLineChange = React.useCallback(
    (lineIdx: number, set: string[]) => {
      const next = lines
        .map((line, idx) => (idx === lineIdx ? set : line))
        .filter((line, idx) => line.length > 0 || idx === lines.length - 1);
      void applyGoals(next, flowSpaceId);
    },
    [applyGoals, flowSpaceId, lines]
  );

  const handleSpaceChange = React.useCallback(
    (value: string) => {
      setFlowSpaceId(value);

      const selected = value ? spaceSelection[value] : null;
      const nextGoals = selected
        ? flowGoals.map((set) => set.filter((id) => selected.has(id)))
        : flowGoals;

      setFlowGoals(nextGoals);
      clearPreviewPlan();
      void applyGoals(nextGoals, value);
    },
    [
      applyGoals,
      clearPreviewPlan,
      flowGoals,
      setFlowGoals,
      setFlowSpaceId,
      spaceSelection,
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
          {lines.map((set, idx) => (
            <div key={idx} className={localStyles.flowGoalSet}>
              {lines.length > 1 && (
                <button
                  type="button"
                  className={localStyles.flowGoalSetRemove}
                  aria-label={t("goals.removeSet")}
                  onClick={() =>
                    void applyGoals(
                      lines.filter((_, i) => i !== idx),
                      flowSpaceId
                    )
                  }
                >
                  <IconClose aria-hidden="true" />
                </button>
              )}
              <GoalSetLine
                context={context}
                onChange={(next) => handleLineChange(idx, next)}
                set={set}
                steps={displaySteps}
                t={t}
              />
              {idx < lines.length - 1 && (
                <div className={localStyles.flowGoalOr}>
                  {t("goals.fallbackTo")}
                  <IconGoalFallback size={16} aria-hidden="true" />
                </div>
              )}
            </div>
          ))}
          <button
            type="button"
            className={localStyles.flowGoalOrButton}
            disabled={lines[lines.length - 1].length === 0}
            aria-label={t("goals.addFallback")}
            title={t("goals.addFallback")}
            onClick={() => setFlowGoals([...lines, []])}
          >
            <IconGoalFallback size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  );
};

export default StepEditorFlowConfiguration;
