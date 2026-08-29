import React from "react";
import Modal from "@/app/components/molecules/Modal";
import { Step, ExecutionPlan } from "@/app/api";
import ScriptConfigEditor from "./ScriptConfigEditor";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { useStepEditorForm } from "./useStepEditorForm";
import { useModalDimensions } from "./useModalDimensions";
import { useT } from "@/app/i18n";
import { useSpaces, useSteps } from "@/app/store/flowStore";
import { useUI } from "@/app/contexts/UIContext";
import { useLabelVocabulary } from "@/app/hooks/useLabelVocabulary";
import { api } from "@/app/api";
import { getFlowPlanAttributeOptions } from "@/utils/flowPlanAttributeOptions";
import {
  IconAttributeLabel,
  IconPredicate,
  IconStepTypeScript,
} from "@/utils/iconRegistry";
import StepEditorBasicFields from "./StepEditorBasicFields";
import StepEditorAttributesSection from "./StepEditorAttributesSection";
import KeyValueTable from "@/app/components/molecules/KeyValueTable";
import StepEditorFlowConfiguration from "./StepEditorFlowConfiguration";
import StepEditorHttpConfiguration from "./StepEditorHttpConfiguration";
import StepEditorFooter from "./StepEditorFooter";
import { predicateLanguageOptions } from "./stepEditorConstants";
import { useScrollFade } from "@/app/hooks/useScrollFade";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  draft?: Step;
}

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
  draft,
}) => {
  const t = useT();
  const steps = useSteps();
  const spaces = useSpaces();
  const { spaceId } = useUI();
  const { labelKeys, valuesForKey } = useLabelVocabulary();
  const defaultLabels = React.useMemo(
    () => spaces.find((space) => space.id === spaceId)?.selector,
    [spaces, spaceId]
  );
  const {
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    handling,
    setHandling,
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    validateJsonDraft,
    getSerializedStepData,
    applyStepData,
    isCreateMode,
    stepId,
    name,
    stepType: formStepType,
    setStepId,
    setName,
    setStepType,
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    labels,
    addLabel,
    updateLabel,
    removeLabel,
    endpoint,
    setEndpoint,
    httpMethod,
    setHttpMethod,
    httpMode,
    setHttpMode,
    healthCheck,
    setHealthCheck,
    compensate,
    setCompensate,
    compensateMethod,
    setCompensateMethod,
    compensateTimeout,
    setCompensateTimeout,
    compensateMode,
    setCompensateMode,
    httpTimeout,
    setHttpTimeout,
    flowGoals,
    setFlowGoals,
    flowCompensate,
    setFlowCompensate,
    flowSpaceId,
    setFlowSpaceId,
  } = useStepEditorForm({ step, onUpdate, onClose, defaultLabels, draft });

  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");
  const [jsonDraft, setJsonDraft] = React.useState("");
  const [flowPreviewPlan, setFlowPreviewPlan] =
    React.useState<ExecutionPlan | null>(null);
  const [flowInitialState, setFlowInitialState] = React.useState("{}");

  const updateFlowPreviewPlan = React.useCallback(
    async (goalSteps: string[], initialState: Record<string, any>) => {
      const plan = await api.getExecutionPlan({ goalSteps, initialState });
      setFlowPreviewPlan(plan);
    },
    []
  );

  const clearFlowPreviewPlan = React.useCallback(() => {
    setFlowPreviewPlan(null);
  }, []);

  const { flowInputOptions, flowOutputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(flowPreviewPlan),
    [flowPreviewPlan]
  );

  const { dimensions, mounted } = useModalDimensions(diagramContainerRef);
  const { scrollRef, showTopFade, showBottomFade } = useScrollFade(
    mounted && editorMode === "basic"
  );

  React.useEffect(() => {
    setEditorMode("basic");
    setJsonDraft(getSerializedStepData());
  }, [getSerializedStepData, step]);

  const handleEditorModeChange = (mode: "basic" | "json") => {
    if (mode === editorMode) return;

    if (mode === "json") {
      setJsonDraft(getSerializedStepData());
      setError(null);
      setEditorMode("json");
      return;
    }

    const jsonError = validateJsonDraft(jsonDraft);
    if (jsonError) {
      setError(jsonError);
      return;
    }

    applyStepData(JSON.parse(jsonDraft) as Step);
    setEditorMode("basic");
  };

  const handleSaveClick = () => {
    if (editorMode === "json") {
      void handleJsonSave(jsonDraft);
      return;
    }
    void handleSave();
  };

  if (!mounted) return null;

  return (
    <Modal
      isOpen
      onClose={onClose}
      width={dimensions.width}
      height={dimensions.height}
      title={
        isCreateMode
          ? t("stepEditor.modalCreateTitle")
          : t("stepEditor.modalEditTitle", { id: stepId })
      }
      footer={
        <StepEditorFooter
          editorMode={editorMode}
          onEditorModeChange={handleEditorModeChange}
          onCancel={onClose}
          onSave={handleSaveClick}
          saving={saving}
          isCreateMode={isCreateMode}
        />
      }
    >
      <>
        <div
          ref={scrollRef}
          className={`${formStyles.formContainer} ${
            editorMode === "json" ? formStyles.formContainerJsonMode : ""
          } ${showTopFade ? formStyles.fadeTop : ""} ${
            showBottomFade ? formStyles.fadeBottom : ""
          }`}
        >
          {editorMode === "basic" ? (
            <>
              <StepEditorBasicFields
                handling={handling}
                isCreateMode={isCreateMode}
                name={name}
                setHandling={setHandling}
                setName={setName}
                setStepId={setStepId}
                setStepType={setStepType}
                stepId={stepId}
                stepType={formStepType}
              />

              <StepEditorAttributesSection
                addAttribute={addAttribute}
                attributes={attributes}
                flowInputOptions={flowInputOptions}
                flowOutputOptions={flowOutputOptions}
                removeAttribute={removeAttribute}
                stepType={formStepType}
                handling={handling}
                updateAttribute={updateAttribute}
              />

              <ScriptConfigEditor
                Icon={IconPredicate}
                label={t("stepEditor.predicateLabel")}
                value={predicate}
                onChange={setPredicate}
                language={predicateLanguage}
                onLanguageChange={setPredicateLanguage}
                languageOptions={predicateLanguageOptions}
                containerClassName={formStyles.predicateEditorContainer}
              />

              {formStepType === "script" ? (
                <ScriptConfigEditor
                  Icon={IconStepTypeScript}
                  label={t("stepEditor.scriptLabel")}
                  value={script}
                  onChange={setScript}
                  language={scriptLanguage}
                  onLanguageChange={setScriptLanguage}
                  containerClassName={formStyles.scriptEditorContainer}
                />
              ) : formStepType === "flow" ? (
                <StepEditorFlowConfiguration
                  clearPreviewPlan={clearFlowPreviewPlan}
                  flowCompensate={flowCompensate}
                  flowGoals={flowGoals}
                  flowInitialState={flowInitialState}
                  flowSpaceId={flowSpaceId}
                  previewPlan={flowPreviewPlan}
                  setFlowCompensate={setFlowCompensate}
                  setFlowGoals={setFlowGoals}
                  setFlowInitialState={setFlowInitialState}
                  setFlowSpaceId={setFlowSpaceId}
                  stepId={stepId}
                  steps={steps}
                  updatePreviewPlan={updateFlowPreviewPlan}
                />
              ) : (
                <StepEditorHttpConfiguration
                  endpoint={endpoint}
                  httpMethod={httpMethod}
                  httpMode={httpMode}
                  healthCheck={healthCheck}
                  compensate={compensate}
                  compensateMethod={compensateMethod}
                  compensateTimeout={compensateTimeout}
                  compensateMode={compensateMode}
                  setCompensateMode={setCompensateMode}
                  httpTimeout={httpTimeout}
                  handling={handling}
                  stepType={formStepType}
                  setEndpoint={setEndpoint}
                  setHttpMethod={setHttpMethod}
                  setHttpMode={setHttpMode}
                  setHealthCheck={setHealthCheck}
                  setCompensate={setCompensate}
                  setCompensateMethod={setCompensateMethod}
                  setCompensateTimeout={setCompensateTimeout}
                  setHttpTimeout={setHttpTimeout}
                />
              )}

              <KeyValueTable
                Icon={IconAttributeLabel}
                label={t("stepEditor.labelsLabel")}
                addLabel={t("stepEditor.addLabel")}
                removeLabel={t("stepEditor.removeLabel")}
                keyPlaceholder={t("stepEditor.labelKeyPlaceholder")}
                valuePlaceholder={t("stepEditor.labelValuePlaceholder")}
                pairs={labels}
                onAdd={addLabel}
                onChange={updateLabel}
                onRemove={removeLabel}
                keySuggestions={labelKeys}
                valueSuggestions={valuesForKey}
              />
            </>
          ) : (
            <div className={formStyles.jsonSection}>
              <div className={formStyles.jsonEditorContainer}>
                <ScriptEditor
                  value={jsonDraft}
                  onChange={setJsonDraft}
                  language="json"
                />
              </div>
            </div>
          )}
        </div>

        {error && (
          <div className={`${formStyles.errorMessage} ${styles.errorBanner}`}>
            {error}
          </div>
        )}
      </>
    </Modal>
  );
};

export default StepEditor;
