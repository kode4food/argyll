import React, { useEffect, createContext, useContext } from "react";
import { createPortal } from "react-dom";
import {
  FileCode2,
  Globe,
  Webhook,
  Trash2,
  Plus,
  Layers,
  Square,
  Workflow,
} from "lucide-react";
import {
  Step,
  AttributeType,
  StepType,
  ExecutionPlan,
  AttributeRole,
} from "@/app/api";
import ScriptConfigEditor from "./ScriptConfigEditor";
import DurationInput from "@/app/components/molecules/DurationInput";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { useStepEditorForm } from "./useStepEditorForm";
import { useModalDimensions } from "./useModalDimensions";
import {
  Attribute,
  getAttributeIconProps,
  parseFlowGoals,
} from "./stepEditorUtils";
import { useT } from "@/app/i18n";
import { useSteps } from "@/app/store/flowStore";
import { useFlowFormStepFiltering } from "../FlowCreateForm/useFlowFormStepFiltering";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";
import { api } from "@/app/api";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
}

interface StepEditingContextValue {
  stepId: string;
  name: string;
  stepType: StepType;
  isCreateMode: boolean;
  setStepId: (value: string) => void;
  setName: (value: string) => void;
  setStepType: (value: StepType) => void;
  attributes: Attribute[];
  addAttribute: () => void;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
  removeAttribute: (id: string) => void;
  cycleAttributeType: (
    id: string,
    currentType: "input" | "optional" | "const" | "output"
  ) => void;
  endpoint: string;
  setEndpoint: (value: string) => void;
  healthCheck: string;
  setHealthCheck: (value: string) => void;
  httpTimeout: number;
  setHttpTimeout: (value: number) => void;
  flowGoals: string;
  setFlowGoals: (value: string) => void;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
}

type FlowInputOption = {
  name: string;
  required: boolean;
};

const ATTRIBUTE_TYPES: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

const StepEditingContext = createContext<StepEditingContextValue | null>(null);

const useStepEditingContext = (): StepEditingContextValue => {
  const ctx = useContext(StepEditingContext);
  if (!ctx) {
    throw new Error(
      "useStepEditingContext must be used within a StepEditor provider"
    );
  }
  return ctx;
};

const BasicFields: React.FC = () => {
  const t = useT();
  const {
    stepId,
    name,
    stepType,
    isCreateMode,
    setStepId,
    setName,
    setStepType,
  } = useStepEditingContext();

  return (
    <div className={formStyles.row}>
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.label}>
          {t("stepEditor.stepIdLabel")}
        </label>
        <input
          type="text"
          value={stepId}
          onChange={(e) => setStepId(e.target.value)}
          className={formStyles.formControl}
          disabled={!isCreateMode}
          placeholder={t("stepEditor.stepIdPlaceholder")}
        />
      </div>
      <div className={`${formStyles.field} ${formStyles.flex2}`}>
        <label className={formStyles.label}>
          {t("stepEditor.stepNameLabel")}
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className={formStyles.formControl}
          placeholder={t("stepEditor.stepNamePlaceholder")}
        />
      </div>
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.label}>{t("stepEditor.typeLabel")}</label>
        <div className={formStyles.typeButtonGroup}>
          {[
            {
              type: "sync" as StepType,
              Icon: Globe,
              label: t("stepEditor.typeSyncLabel"),
              title: t("stepEditor.typeSyncTitle"),
            },
            {
              type: "async" as StepType,
              Icon: Webhook,
              label: t("stepEditor.typeAsyncLabel"),
              title: t("stepEditor.typeAsyncTitle"),
            },
            {
              type: "script" as StepType,
              Icon: FileCode2,
              label: t("stepEditor.typeScriptLabel"),
              title: t("stepEditor.typeScriptTitle"),
            },
            {
              type: "flow" as StepType,
              Icon: Workflow,
              label: t("stepEditor.typeFlowLabel"),
              title: t("stepEditor.typeFlowTitle"),
            },
          ].map(({ type, Icon, label, title }) => (
            <button
              key={type}
              type="button"
              onClick={(e) => {
                setStepType(type);
                e.currentTarget.blur();
              }}
              className={`${formStyles.typeButton} ${stepType === type ? formStyles.typeButtonActive : ""}`}
              title={title}
            >
              <Icon className={styles.iconSm} />
              <span>{label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};

const AttributesSection: React.FC = () => {
  const t = useT();
  const {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    cycleAttributeType,
    stepType,
    flowInputOptions,
    flowOutputOptions,
  } = useStepEditingContext();

  const flowInputList = flowInputOptions;
  const flowOutputList = flowOutputOptions;

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.attributesLabel")}
        </label>
        <button
          onClick={addAttribute}
          className={`${formStyles.iconButton} ${formStyles.addButtonStyle}`}
          title={t("stepEditor.addAttribute")}
        >
          <Plus className={styles.iconMd} />
        </button>
      </div>
      <div className={formStyles.argList}>
        {attributes.length === 0 && (
          <div
            className={`${formStyles.attrRow} ${formStyles.attrPlaceholder}`}
            aria-hidden
          >
            <div className={formStyles.attrRowInputs}>
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderIcon}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSelect}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderInput}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSmall}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderButton}`}
              />
            </div>
            <div className={formStyles.placeholderHint}>
              {t("stepEditor.attributesHint")}
            </div>
          </div>
        )}
        {attributes.map((attr) => (
          <div key={attr.id} className={formStyles.attrRow}>
            <div className={formStyles.attrRowInputs}>
              <button
                type="button"
                onClick={() => cycleAttributeType(attr.id, attr.attrType)}
                className={`${formStyles.iconButton} ${formStyles.attrIconButtonStyle}`}
                title={t("stepEditor.cycleAttributeType", {
                  type: attr.attrType,
                })}
              >
                {(() => {
                  const { Icon, className } = getAttributeIconProps(
                    attr.attrType
                  );
                  return <Icon className={`${styles.iconMd} ${className}`} />;
                })()}
              </button>
              <select
                value={attr.dataType}
                onChange={(e) =>
                  updateAttribute(attr.id, "dataType", e.target.value)
                }
                className={formStyles.argType}
              >
                {ATTRIBUTE_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
              <input
                type="text"
                value={attr.name}
                onChange={(e) =>
                  updateAttribute(attr.id, "name", e.target.value)
                }
                placeholder={t("stepEditor.attributeNamePlaceholder")}
                className={formStyles.argInput}
              />
              {(attr.attrType === "optional" || attr.attrType === "const") && (
                <input
                  type="text"
                  value={attr.defaultValue || ""}
                  onChange={(e) =>
                    updateAttribute(attr.id, "defaultValue", e.target.value)
                  }
                  placeholder={t("stepEditor.attributeDefaultPlaceholder")}
                  className={formStyles.argInput}
                  title={t("stepEditor.attributeDefaultTitle")}
                />
              )}
              {attr.attrType !== "output" &&
                attr.dataType === AttributeType.Array && (
                  <div className={formStyles.forEachToggleGroup}>
                    <button
                      type="button"
                      onClick={(e) => {
                        updateAttribute(attr.id, "forEach", false);
                        e.currentTarget.blur();
                      }}
                      className={`${formStyles.forEachToggle} ${!attr.forEach ? formStyles.forEachToggleActive : ""}`}
                      title={t("stepEditor.arraySingleTitle")}
                    >
                      <Square className={styles.iconSm} />
                      <span>{t("stepEditor.arraySingleLabel")}</span>
                    </button>
                    <button
                      type="button"
                      onClick={(e) => {
                        updateAttribute(attr.id, "forEach", true);
                        e.currentTarget.blur();
                      }}
                      className={`${formStyles.forEachToggle} ${attr.forEach ? formStyles.forEachToggleActive : ""}`}
                      title={t("stepEditor.arrayMultiTitle")}
                    >
                      <Layers className={styles.iconSm} />
                      <span>{t("stepEditor.arrayMultiLabel")}</span>
                    </button>
                  </div>
                )}
              {stepType === "flow" && (
                <select
                  value={attr.flowMap || ""}
                  onChange={(e) =>
                    updateAttribute(attr.id, "flowMap", e.target.value)
                  }
                  className={formStyles.flowMapSelect}
                  disabled={
                    attr.attrType === "output"
                      ? flowOutputList.length === 0
                      : flowInputList.length === 0
                  }
                >
                  <option value="">{t("stepEditor.flowMapPlaceholder")}</option>
                  {attr.attrType === "output"
                    ? flowOutputList.map((option) => (
                        <option key={option} value={option}>
                          {option}
                        </option>
                      ))
                    : flowInputList.map((option) => (
                        <option
                          key={option.name}
                          value={option.name}
                          className={
                            option.required
                              ? formStyles.flowMapOptionRequired
                              : undefined
                          }
                        >
                          {option.name}
                        </option>
                      ))}
                </select>
              )}
              <button
                onClick={() => removeAttribute(attr.id)}
                className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
                title={t("stepEditor.removeAttribute")}
              >
                <Trash2 className={styles.iconSm} />
              </button>
            </div>
            {attr.validationError && (
              <div className={formStyles.attrValidationError}>
                {attr.validationError}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

const FlowConfiguration: React.FC<{
  previewPlan: ExecutionPlan | null;
  flowInitialState: string;
  setFlowInitialState: (value: string) => void;
  updatePreviewPlan: (
    goalSteps: string[],
    initialState: Record<string, any>
  ) => Promise<void>;
  clearPreviewPlan: () => void;
}> = ({
  previewPlan,
  flowInitialState,
  setFlowInitialState,
  updatePreviewPlan,
  clearPreviewPlan,
}) => {
  const t = useT();
  const steps = useSteps();
  const { flowGoals, setFlowGoals, stepId } = useStepEditingContext();

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

  const { included, satisfied, missingByStep } = useFlowFormStepFiltering(
    displaySteps,
    flowInitialState,
    previewPlan
  );

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
      goalList,
      flowInitialState,
      sortedSteps,
      setFlowInitialState,
      setFlowGoals,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.flowGoalsLabel")}
        </label>
      </div>
      <div className={formStyles.flowGoalList}>
        {displaySteps.map((step) => {
          const isSelected = goalList.includes(step.id);
          const isIncludedByOthers = included.has(step.id) && !isSelected;
          const isSatisfiedByState = satisfied.has(step.id) && !isSelected;
          const missingRequired = missingByStep.get(step.id) || [];
          const isMissing = missingRequired.length > 0;
          const isDisabled = isIncludedByOthers || isSatisfiedByState;

          const tooltipText = isIncludedByOthers
            ? t("flowCreate.tooltipAlreadyIncluded")
            : isSatisfiedByState
              ? t("flowCreate.tooltipSatisfiedByState")
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
                if (isDisabled) return;
                void handleGoalToggle(step.id);
              }}
              disabled={isDisabled}
              className={`${formStyles.flowGoalChip} ${
                isSelected ? formStyles.flowGoalChipSelected : ""
              } ${isIncludedByOthers ? formStyles.flowGoalChipIncluded : ""} ${
                isDisabled ? formStyles.flowGoalChipDisabled : ""
              }`}
            >
              {step.id}
              {isIncludedByOthers && (
                <span className={formStyles.flowGoalChipCheck}>✓</span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
};

const HttpConfiguration: React.FC = () => {
  const t = useT();
  const {
    endpoint,
    httpTimeout,
    healthCheck,
    setEndpoint,
    setHttpTimeout,
    setHealthCheck,
  } = useStepEditingContext();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.httpConfigLabel")}
        </label>
      </div>
      <div className={formStyles.httpFields}>
        <div className={formStyles.row}>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.label}>
              {t("stepEditor.endpointLabel")}
            </label>
            <input
              type="text"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder={t("stepEditor.endpointPlaceholder")}
              className={formStyles.formControl}
            />
          </div>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.timeoutLabel")}
            </label>
            <DurationInput value={httpTimeout} onChange={setHttpTimeout} />
          </div>
        </div>
        <div className={formStyles.field}>
          <label className={formStyles.label}>
            {t("stepEditor.healthCheckLabel")}
          </label>
          <input
            type="text"
            value={healthCheck}
            onChange={(e) => setHealthCheck(e.target.value)}
            placeholder={t("stepEditor.healthCheckPlaceholder")}
            className={formStyles.formControl}
          />
        </div>
      </div>
    </div>
  );
};

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
}) => {
  const t = useT();
  const {
    stepId,
    stepType,
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    saving,
    error,
    handleSave,
    isCreateMode,
    contextValue,
  } = useStepEditorForm(step, onUpdate, onClose);

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

  const { flowInputOptions, flowOutputOptions } = React.useMemo(() => {
    if (!flowPreviewPlan?.steps) {
      return { flowInputOptions: [], flowOutputOptions: [] };
    }

    const inputMap = new Map<string, FlowInputOption>();
    const outputSet = new Set<string>();

    Object.values(flowPreviewPlan.steps).forEach((planStep) => {
      Object.entries(planStep.attributes || {}).forEach(([name, spec]) => {
        if (spec.role === AttributeRole.Required) {
          inputMap.set(name, { name, required: true });
        }
        if (spec.role === AttributeRole.Optional && !inputMap.has(name)) {
          inputMap.set(name, { name, required: false });
        }
        if (spec.role === AttributeRole.Output) {
          outputSet.add(name);
        }
      });
    });

    return {
      flowInputOptions: Array.from(inputMap.values()).sort((a, b) =>
        a.name.localeCompare(b.name)
      ),
      flowOutputOptions: Array.from(outputSet).sort((a, b) =>
        a.localeCompare(b)
      ),
    };
  }, [flowPreviewPlan]);

  const extendedContextValue = React.useMemo(
    () => ({
      ...contextValue,
      flowInputOptions,
      flowOutputOptions,
    }),
    [contextValue, flowInputOptions, flowOutputOptions]
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

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!mounted) return null;

  const modalContent = (
    <StepEditingContext.Provider value={extendedContextValue}>
      <div className={styles.backdrop} onClick={handleBackdropClick}>
        <div
          className={styles.content}
          style={{
            width: `${dimensions.width}px`,
            minHeight: `${dimensions.minHeight}px`,
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className={styles.header}>
            <h2 className={styles.title}>
              {isCreateMode
                ? t("stepEditor.modalCreateTitle")
                : t("stepEditor.modalEditTitle", { id: stepId })}
            </h2>
          </div>

          <div className={styles.body}>
            <div className={formStyles.formContainer}>
              {/* Basic Fields */}
              <BasicFields />

              {stepType === "flow" && (
                <FlowConfiguration
                  previewPlan={flowPreviewPlan}
                  flowInitialState={flowInitialState}
                  setFlowInitialState={setFlowInitialState}
                  updatePreviewPlan={updateFlowPreviewPlan}
                  clearPreviewPlan={clearFlowPreviewPlan}
                />
              )}

              {/* Unified Attributes Section */}
              <AttributesSection />

              {/* Predicate */}
              <ScriptConfigEditor
                label={t("stepEditor.predicateLabel")}
                value={predicate}
                onChange={setPredicate}
                language={predicateLanguage}
                onLanguageChange={setPredicateLanguage}
                containerClassName={formStyles.predicateEditorContainer}
              />

              {/* Type-Specific Configuration */}
              {stepType === "script" ? (
                <ScriptConfigEditor
                  label={t("stepEditor.scriptLabel")}
                  value={script}
                  onChange={setScript}
                  language={scriptLanguage}
                  onLanguageChange={setScriptLanguage}
                  containerClassName={formStyles.scriptEditorContainer}
                />
              ) : stepType === "flow" ? null : (
                <HttpConfiguration />
              )}

              {error && <div className={formStyles.errorMessage}>{error}</div>}
            </div>
          </div>

          <div className={styles.footer}>
            <button
              onClick={onClose}
              disabled={saving}
              className={styles.buttonCancel}
            >
              {t("stepEditor.cancel")}
            </button>
            <button
              onClick={handleSave}
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
    </StepEditingContext.Provider>
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
