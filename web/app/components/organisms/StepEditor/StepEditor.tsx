import React, { useEffect } from "react";
import { createPortal } from "react-dom";
import { Step, ExecutionPlan } from "@/app/api";
import ScriptConfigEditor from "./ScriptConfigEditor";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { useStepEditorForm } from "./useStepEditorForm";
import { useModalDimensions } from "./useModalDimensions";
import { useT } from "@/app/i18n";
import { useSteps } from "@/app/store/flowStore";
import { api } from "@/app/api";
import { getFlowPlanAttributeOptions } from "@/utils/flowPlanAttributeOptions";
import StepEditorBasicFields from "./StepEditorBasicFields";
import StepEditorAttributesSection from "./StepEditorAttributesSection";
import StepEditorFlowConfiguration from "./StepEditorFlowConfiguration";
import StepEditorHttpConfiguration from "./StepEditorHttpConfiguration";
import StepEditorHeader from "./StepEditorHeader";
import StepEditorFooter from "./StepEditorFooter";
import { predicateLanguageOptions } from "./stepEditorConstants";
import { useScrollFade } from "@/app/hooks/useScrollFade";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
}) => {
  const t = useT();
  const steps = useSteps();
  const {
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    memoizable,
    setMemoizable,
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
    endpoint,
    setEndpoint,
    httpMethod,
    setHttpMethod,
    healthCheck,
    setHealthCheck,
    compensate,
    setCompensate,
    compensateMethod,
    setCompensateMethod,
    compensateTimeout,
    setCompensateTimeout,
    httpTimeout,
    setHttpTimeout,
    flowGoals,
    setFlowGoals,
    flowCompensate,
    setFlowCompensate,
  } = useStepEditorForm(step, onUpdate, onClose);

  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");
  const [jsonDraft, setJsonDraft] = React.useState("");
  const [flowPreviewPlan, setFlowPreviewPlan] =
    React.useState<ExecutionPlan | null>(null);
  const [flowInitialState, setFlowInitialState] = React.useState("{}");

  const updateFlowPreviewPlan = React.useCallback(
    async (goalSteps: string[], initialState: Record<string, any>) => {
      const plan = await api.getExecutionPlan(goalSteps, initialState);
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
  const { scrollRef, showTopFade, showBottomFade } = useScrollFade(mounted);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [onClose]);

  useEffect(() => {
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

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) onClose();
  };

  const handleSaveClick = () => {
    if (editorMode === "json") {
      void handleJsonSave(jsonDraft);
      return;
    }
    void handleSave();
  };

  if (!mounted) return null;

  // Compensate only reaches the payload for HTTP steps, so a leftover value
  // from an earlier step type must not lock memoizable
  const isHttpStep = formStepType === "sync" || formStepType === "async";
  const isMemoizableDisabled = isHttpStep && compensate.trim() !== "";

  const modalContent = (
    <div className={styles.backdrop} onClick={handleBackdropClick}>
      <div
        className={styles.content}
        data-ui-overlay="modal"
        style={{
          width: `${dimensions.width}px`,
          height: `${dimensions.height}px`,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <StepEditorHeader
          isCreateMode={isCreateMode}
          stepId={stepId}
          memoizable={memoizable}
          memoizableDisabled={isMemoizableDisabled}
          onMemoizableChange={setMemoizable}
        />

        <div className={styles.body}>
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
                  isCreateMode={isCreateMode}
                  name={name}
                  setName={setName}
                  setStepId={setStepId}
                  setStepType={setStepType}
                  stepId={stepId}
                  stepType={formStepType}
                />

                {formStepType === "flow" && (
                  <StepEditorFlowConfiguration
                    clearPreviewPlan={clearFlowPreviewPlan}
                    flowCompensate={flowCompensate}
                    flowGoals={flowGoals}
                    flowInitialState={flowInitialState}
                    previewPlan={flowPreviewPlan}
                    setFlowCompensate={setFlowCompensate}
                    setFlowGoals={setFlowGoals}
                    setFlowInitialState={setFlowInitialState}
                    stepId={stepId}
                    steps={steps}
                    updatePreviewPlan={updateFlowPreviewPlan}
                  />
                )}

                <StepEditorAttributesSection
                  addAttribute={addAttribute}
                  attributes={attributes}
                  flowInputOptions={flowInputOptions}
                  flowOutputOptions={flowOutputOptions}
                  removeAttribute={removeAttribute}
                  stepType={formStepType}
                  updateAttribute={updateAttribute}
                />

                <ScriptConfigEditor
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
                    label={t("stepEditor.scriptLabel")}
                    value={script}
                    onChange={setScript}
                    language={scriptLanguage}
                    onLanguageChange={setScriptLanguage}
                    containerClassName={formStyles.scriptEditorContainer}
                  />
                ) : formStepType === "flow" ? null : (
                  <StepEditorHttpConfiguration
                    endpoint={endpoint}
                    httpMethod={httpMethod}
                    healthCheck={healthCheck}
                    compensate={compensate}
                    compensateMethod={compensateMethod}
                    compensateTimeout={compensateTimeout}
                    httpTimeout={httpTimeout}
                    memoizable={memoizable}
                    setEndpoint={setEndpoint}
                    setHttpMethod={setHttpMethod}
                    setHealthCheck={setHealthCheck}
                    setCompensate={setCompensate}
                    setCompensateMethod={setCompensateMethod}
                    setCompensateTimeout={setCompensateTimeout}
                    setHttpTimeout={setHttpTimeout}
                  />
                )}
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
        </div>

        {error && (
          <div className={`${formStyles.errorMessage} ${styles.errorBanner}`}>
            {error}
          </div>
        )}

        <StepEditorFooter
          editorMode={editorMode}
          onEditorModeChange={handleEditorModeChange}
          onCancel={onClose}
          onSave={handleSaveClick}
          saving={saving}
          isCreateMode={isCreateMode}
        />
      </div>
    </div>
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
