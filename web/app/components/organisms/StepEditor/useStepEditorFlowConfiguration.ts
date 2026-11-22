import React from "react";
import { ExecutionPlan, GoalSets, Step } from "@/app/api";
import { useT } from "@/app/i18n";
import {
  useSpaceOptions,
  useSpaceScopedSteps,
} from "@/app/hooks/useSpaceOptions";
import { useSpaceSelection } from "@/app/store/flowStore";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";
import { useFlowFormStepFiltering } from "@/app/hooks/useFlowFormStepFiltering";

export interface StepEditorFlowConfigurationOptions {
  clearPreviewPlan: () => void;
  flowGoals: GoalSets;
  flowInitialState: string;
  flowSpaceId: string;
  previewPlan: ExecutionPlan | null;
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

const useFlowSpaceScope = (
  steps: Step[],
  flowSpaceId: string,
  stepId: string
) => {
  const t = useT();
  const spaceSelection = useSpaceSelection();
  const sortedSteps = React.useMemo(
    () => [...steps].sort((a, b) => a.name.localeCompare(b.name)),
    [steps]
  );
  const spaceOptions = useSpaceOptions(t("flowCreate.spaceAll"));
  const scopedSteps = useSpaceScopedSteps(sortedSteps, flowSpaceId || null);
  const displaySteps = React.useMemo(
    () => scopedSteps.filter((step) => step.id !== stepId),
    [scopedSteps, stepId]
  );
  return { spaceSelection, sortedSteps, spaceOptions, displaySteps };
};

export const useStepEditorFlowConfiguration = ({
  clearPreviewPlan,
  flowGoals,
  flowInitialState,
  flowSpaceId,
  previewPlan,
  setFlowGoals,
  setFlowInitialState,
  setFlowSpaceId,
  stepId,
  steps,
  updatePreviewPlan,
}: StepEditorFlowConfigurationOptions) => {
  const lines = React.useMemo<GoalSets>(
    () => (flowGoals.length > 0 ? flowGoals : [[]]),
    [flowGoals]
  );
  const hasGoals = flowGoals.some((set) => set.length > 0);
  const initializedGoalsRef = React.useRef(false);
  const { spaceSelection, sortedSteps, spaceOptions, displaySteps } =
    useFlowSpaceScope(steps, flowSpaceId, stepId);

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

  const handleRemoveSet = (idx: number) =>
    void applyGoals(
      lines.filter((_, i) => i !== idx),
      flowSpaceId
    );
  const handleAddFallback = () => setFlowGoals([...lines, []]);

  const committedLines = lines.slice(0, -1);
  const activeLine = lines[lines.length - 1];
  const activeLineIdx = lines.length - 1;

  return {
    spaceOptions,
    displaySteps,
    context,
    committedLines,
    activeLine,
    activeLineIdx,
    handleLineChange,
    handleSpaceChange,
    handleRemoveSet,
    handleAddFallback,
  };
};
