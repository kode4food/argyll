import React from "react";
import { useUI } from "@/app/contexts/UIContext";
import { fromGoalSets, Step } from "@/app/api";
import { useFlowCreation } from "@/app/hooks/useFlowCreation";
import { useScrollFade } from "@/app/hooks/useScrollFade";
import { useT } from "@/app/i18n";
import {
  useSpaceOptions,
  useSpaceScopedSteps,
} from "@/app/hooks/useSpaceOptions";
import { getFlowPlanAttributeOptions } from "@/utils/flowPlanAttributeOptions";
import { generateFlowId } from "@/utils/flowUtils";
import { nonEmptyGoalSets } from "@/utils/goalSets";
import { IconSpace } from "@/utils/iconRegistry";
import { sortStepsByType } from "@/utils/stepUtils";
import { validateJsonString } from "./flowFormUtils";
import { useFlowFormStepFiltering } from "@/app/hooks/useFlowFormStepFiltering";
import { useFlowInputDrafts } from "./useFlowInputDrafts";

const useSpaceScope = (steps: Step[], spaceId: string | null) => {
  const t = useT();
  const spaceOptions = useSpaceOptions(t("flowCreate.spaceAll"), IconSpace);
  const scopedSteps = useSpaceScopedSteps(steps, spaceId);
  const sortedSteps = React.useMemo(
    () => sortStepsByType(scopedSteps),
    [scopedSteps]
  );
  return { spaceOptions, scopedSteps, sortedSteps };
};

export const useFlowCreateForm = (onCreateStep?: (draft?: Step) => void) => {
  const {
    newID,
    setNewID,
    setIDManuallyEdited,
    handleStepChange,
    initialState,
    setInitialState,
    creating,
    handleCreateFlow,
    steps,
  } = useFlowCreation();
  const {
    previewPlan,
    goalSteps,
    setHoveredGoalSet,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
    spaceId,
    setSpaceId,
  } = useUI();
  const t = useT();

  const [jsonError, setJsonError] = React.useState<string | null>(null);

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  const { spaceOptions, scopedSteps, sortedSteps } = useSpaceScope(
    steps,
    spaceId
  );

  const sidebarFade = useScrollFade();

  const stepFiltering = useFlowFormStepFiltering(
    steps,
    initialState,
    previewPlan
  );
  const planGoals = React.useMemo(
    () => nonEmptyGoalSets(goalSteps),
    [goalSteps]
  );
  const { flowInputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(previewPlan, steps),
    [previewPlan, steps]
  );
  const emptyAttributesLabel =
    planGoals.length === 0
      ? t("flowCreate.noGoalsSelected")
      : t("flowCreate.noPotentialInputs");

  const inputDrafts = useFlowInputDrafts({
    flowInputOptions,
    initialState,
    setInitialState,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  });

  // Excludes the flow ID: disabling its input on an empty ID would make the
  // ID impossible to type
  const isStartSectionDisabled =
    creating ||
    planGoals.length === 0 ||
    (inputDrafts.editorMode === "json" && jsonError !== null);

  const handleCreateStep = React.useCallback(
    (fromGoals: boolean) => {
      onCreateStep?.(
        fromGoals
          ? {
              id: "",
              name: "",
              type: "flow",
              attributes: {},
              flow: {
                goals: fromGoalSets(planGoals),
                ...(spaceId && { space_id: spaceId }),
              },
            }
          : undefined
      );
    },
    [planGoals, onCreateStep, spaceId]
  );

  const handleSpaceChange = (value: string) => setSpaceId(value || null);
  const handleFlowIdChange = (value: string) => {
    setNewID(value);
    setIDManuallyEdited(true);
  };
  const handleGenerateId = () => {
    setNewID(generateFlowId());
    setIDManuallyEdited(false);
  };

  return {
    newID,
    handleStepChange,
    initialState,
    setInitialState,
    creating,
    handleCreateFlow,
    goalSteps,
    setHoveredGoalSet,
    setFocusedPreviewAttribute,
    spaceId,
    jsonError,
    spaceOptions,
    scopedSteps,
    sortedSteps,
    sidebarFade,
    stepFiltering,
    flowInputOptions,
    emptyAttributesLabel,
    inputDrafts,
    isStartSectionDisabled,
    handleCreateStep,
    handleSpaceChange,
    handleFlowIdChange,
    handleGenerateId,
  };
};
