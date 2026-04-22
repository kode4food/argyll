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
import { PREDICATE_LANGUAGE_OPTIONS } from "./stepEditorConstants";

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
    applyStepDataToForm,
    isCreateMode,
    contextValue: {
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
      cycleAttributeType,
      endpoint,
      setEndpoint,
      httpMethod,
      setHttpMethod,
      healthCheck,
      setHealthCheck,
      httpTimeout,
      setHttpTimeout,
      flowGoals,
      setFlowGoals,
    },
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

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [onClose]);

  useEffect(() => {
    setEditorMode("basic");
    setJsonDraft(getSerializedStepData());
  }, [getSerializedStepData, step]);

  const handleEditorModeChange = (mode: "basic" | "json") => {
    if (mode === editorMode) {
      return;
    }

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

    applyStepDataToForm(JSON.parse(jsonDraft) as Step);
    setEditorMode("basic");
  };

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!mounted) return null;

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
        <div className={styles.header}>
          <h2 className={styles.title}>
            {isCreateMode
              ? t("stepEditor.modalCreateTitle")
              : t("stepEditor.modalEditTitle", { id: stepId })}
          </h2>
          <div className={styles.headerControls}>
            <label
              className={styles.headerCheckboxLabel}
              title={t("stepEditor.memoizableTitle")}
            >
              <span>{t("stepEditor.memoizableLabel")}</span>
              <input
                type="checkbox"
                checked={memoizable}
                onChange={(e) => setMemoizable(e.target.checked)}
                className={styles.headerCheckbox}
              />
            </label>
          </div>
        </div>

        <div className={styles.body}>
          <div
            className={`${formStyles.formContainer} ${editorMode === "json" ? formStyles.formContainerJsonMode : ""}`}
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
                    flowGoals={flowGoals}
                    flowInitialState={flowInitialState}
                    previewPlan={flowPreviewPlan}
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
                  cycleAttributeType={cycleAttributeType}
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
                  languageOptions={PREDICATE_LANGUAGE_OPTIONS}
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
                    httpTimeout={httpTimeout}
                    setEndpoint={setEndpoint}
                    setHttpMethod={setHttpMethod}
                    setHealthCheck={setHealthCheck}
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

        <div className={styles.footer}>
          <div className={styles.footerControls}>
            <div className={formStyles.editorModeToggleGroup}>
              <button
                type="button"
                className={`${formStyles.editorModeToggle} ${editorMode === "basic" ? formStyles.editorModeToggleActive : ""}`}
                onClick={() => handleEditorModeChange("basic")}
              >
                {t("stepEditor.modeBasic")}
              </button>
              <button
                type="button"
                className={`${formStyles.editorModeToggle} ${editorMode === "json" ? formStyles.editorModeToggleActive : ""}`}
                onClick={() => handleEditorModeChange("json")}
              >
                {t("stepEditor.modeJson")}
              </button>
            </div>
          </div>
          <div className={styles.footerButtons}>
            <button
              onClick={onClose}
              disabled={saving}
              className={styles.buttonCancel}
            >
              {t("stepEditor.cancel")}
            </button>
            <button
              onClick={() => {
                if (editorMode === "json") {
                  void handleJsonSave(jsonDraft);
                  return;
                }
                void handleSave();
              }}
              disabled={saving}
              className={styles.buttonSave}
            >
              {saving
                ? isCreateMode
                  ? t("stepEditor.creating")
                  : t("stepEditor.saving")
                : isCreateMode
                  ? t("stepEditor.create")
                  : t("stepEditor.save")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
