import React from "react";
import { Step, ExecutionPlan, GoalSets } from "@/app/api";
import { api } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import { useScrollFade } from "@/app/hooks/useScrollFade";
import { useTagVocabulary } from "@/app/hooks/useTagVocabulary";
import { useT } from "@/app/i18n";
import { useSpaces, useSteps } from "@/app/store/flowStore";
import { getFlowPlanAttributeOptions } from "@/utils/flowPlanAttributeOptions";
import type { FlowPreview } from "./StepEditorTypeConfiguration";
import { useAttributeCompletions } from "./useAttributeCompletions";
import { useModalDimensions } from "./useModalDimensions";
import { useStepEditorForm } from "./useStepEditorForm";
import { parseJsonStep } from "./useStepPersistence";

export interface StepEditorOptions {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  draft?: Step;
}

type EditorMode = "basic" | "json";

const useFlowPreview = () => {
  const [flowPreviewPlan, setFlowPreviewPlan] =
    React.useState<ExecutionPlan | null>(null);
  const [flowInitialState, setFlowInitialState] = React.useState("{}");

  const updateFlowPreviewPlan = React.useCallback(
    async (
      goalSteps: GoalSets,
      initialState: Record<string, any>,
      previewSpaceId?: string
    ) => {
      const plan = await api.getExecutionPlan({
        goalSteps,
        initialState,
        spaceId: previewSpaceId,
      });
      setFlowPreviewPlan(plan);
    },
    []
  );

  const clearFlowPreviewPlan = React.useCallback(() => {
    setFlowPreviewPlan(null);
  }, []);

  const flowPreview: FlowPreview = {
    previewPlan: flowPreviewPlan,
    flowInitialState,
    setFlowInitialState,
    updatePreviewPlan: updateFlowPreviewPlan,
    clearPreviewPlan: clearFlowPreviewPlan,
  };

  const { flowInputOptions, flowOutputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(flowPreviewPlan),
    [flowPreviewPlan]
  );

  return { flowPreview, flowInputOptions, flowOutputOptions };
};

export const useStepEditor = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
  draft,
}: StepEditorOptions) => {
  const t = useT();
  const steps = useSteps();
  const spaces = useSpaces();
  const { spaceId } = useUI();
  const tagVocabulary = useTagVocabulary();
  const defaultTags = React.useMemo(
    // A new Step only has to satisfy one alternative, so the first is enough
    () => spaces.find((space) => space.id === spaceId)?.qbe?.[0],
    [spaces, spaceId]
  );
  const form = useStepEditorForm({
    step,
    onUpdate,
    onClose,
    defaultTags,
    draft,
  });
  const {
    setError,
    handleSave,
    handleJsonSave,
    getSerializedStepData,
    applyStepData,
    isCreateMode,
    stepId,
  } = form;

  const [editorMode, setEditorMode] = React.useState<EditorMode>("basic");
  const [jsonDraft, setJsonDraft] = React.useState("");
  const [initialData] = React.useState(getSerializedStepData);
  const currentData =
    editorMode === "json" ? jsonDraft : getSerializedStepData();
  // An unsaved draft has changes even before the user edits it
  const isDirty =
    (isCreateMode && draft !== undefined) || currentData !== initialData;
  const { predicateCompletions } = useAttributeCompletions(form.attributes);
  const { flowPreview, flowInputOptions, flowOutputOptions } = useFlowPreview();

  const { dimensions, mounted } = useModalDimensions(diagramContainerRef);
  const { scrollRef, showTopFade, showBottomFade } = useScrollFade(
    mounted && editorMode === "basic"
  );

  React.useEffect(() => {
    setEditorMode("basic");
    setJsonDraft(getSerializedStepData());
  }, [getSerializedStepData, step]);

  const handleEditorModeChange = (mode: EditorMode) => {
    if (mode === editorMode) return;

    if (mode === "json") {
      setJsonDraft(getSerializedStepData());
      setError(null);
      setEditorMode("json");
      return;
    }

    const result = parseJsonStep(jsonDraft, t);
    if ("error" in result) {
      setError(result.error);
      return;
    }

    applyStepData(result.step);
    setError(null);
    setEditorMode("basic");
  };

  const handleSaveClick = () => {
    if (editorMode === "json") {
      void handleJsonSave(jsonDraft);
      return;
    }
    void handleSave();
  };

  const title = isCreateMode
    ? t("stepEditor.modalCreateTitle")
    : t("stepEditor.modalEditTitle", { id: stepId });

  return {
    form,
    steps,
    tagVocabulary,
    editorMode,
    jsonDraft,
    setJsonDraft,
    predicateCompletions,
    flowPreview,
    flowInputOptions,
    flowOutputOptions,
    dimensions,
    mounted,
    scrollRef,
    showTopFade,
    showBottomFade,
    handleEditorModeChange,
    handleSaveClick,
    isDirty,
    title,
  };
};
