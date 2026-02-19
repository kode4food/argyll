import React, { useEffect, createContext, useContext } from "react";
import { createPortal } from "react-dom";
import {
  IconAdd,
  IconArrayMultiple,
  IconArraySingle,
  IconExpandDown,
  IconExpandUp,
  IconRemove,
} from "@/utils/iconRegistry";
import {
  Step,
  AttributeType,
  StepType,
  ExecutionPlan,
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";
import ScriptConfigEditor from "./ScriptConfigEditor";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
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
import { getStepTypeIcon } from "@/utils/iconRegistry";
import {
  FlowInputOption,
  getFlowPlanAttributeOptions,
} from "@/utils/flowPlanAttributeOptions";

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
  memoizable: boolean;
  setMemoizable: (value: boolean) => void;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
}

const ATTRIBUTE_TYPES: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

const PREDICATE_LANGUAGE_OPTIONS = [
  { value: SCRIPT_LANGUAGE_ALE, labelKey: "script.language.ale" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
  { value: SCRIPT_LANGUAGE_JPATH, labelKey: "script.language.jpath" },
];

const MAPPING_LANGUAGE_OPTIONS = [
  { value: SCRIPT_LANGUAGE_ALE, labelKey: "script.language.ale" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
  { value: SCRIPT_LANGUAGE_JPATH, labelKey: "script.language.jpath" },
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
              label: t("stepEditor.typeSyncLabel"),
              title: t("stepEditor.typeSyncTitle"),
            },
            {
              type: "async" as StepType,
              label: t("stepEditor.typeAsyncLabel"),
              title: t("stepEditor.typeAsyncTitle"),
            },
            {
              type: "script" as StepType,
              label: t("stepEditor.typeScriptLabel"),
              title: t("stepEditor.typeScriptTitle"),
            },
            {
              type: "flow" as StepType,
              label: t("stepEditor.typeFlowLabel"),
              title: t("stepEditor.typeFlowTitle"),
            },
          ].map(({ type, label, title }) => {
            const Icon = getStepTypeIcon(type);
            return (
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
            );
          })}
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
  const [expandedMappingAttributeID, setExpandedMappingAttributeID] =
    React.useState<string | null>(null);

  const flowInputList = flowInputOptions;
  const flowOutputList = flowOutputOptions;
  const usedInputMappings = new Map<string, string>();
  const usedOutputMappings = new Map<string, string>();

  attributes.forEach((attr) => {
    const mappingName = attr.mappingName?.trim();
    if (!mappingName) {
      return;
    }
    if (attr.attrType === "output") {
      usedOutputMappings.set(mappingName, attr.id);
      return;
    }
    usedInputMappings.set(mappingName, attr.id);
  });

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
          <IconAdd className={styles.iconMd} />
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
        {attributes.map((attr) => {
          const isMappingExpanded =
            expandedMappingAttributeID === attr.id && attr.attrType !== "const";
          const hasMappingConfigured = Boolean(
            attr.mappingName?.trim() || attr.mappingScript?.trim()
          );

          return (
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
                {(attr.attrType === "optional" ||
                  attr.attrType === "const") && (
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
                        <IconArraySingle className={styles.iconSm} />
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
                        <IconArrayMultiple className={styles.iconSm} />
                        <span>{t("stepEditor.arrayMultiLabel")}</span>
                      </button>
                    </div>
                  )}
                {attr.attrType !== "const" && (
                  <button
                    type="button"
                    onClick={() =>
                      setExpandedMappingAttributeID((current) =>
                        current === attr.id ? null : attr.id
                      )
                    }
                    className={`${formStyles.iconButton} ${formStyles.mappingExpandButton} ${
                      hasMappingConfigured
                        ? formStyles.mappingExpandButtonActive
                        : ""
                    }`}
                    title={t("stepEditor.mappingLabel")}
                    aria-label={`${t("stepEditor.mappingLabel")} ${attr.name || attr.id}`}
                  >
                    {isMappingExpanded ? (
                      <IconExpandUp className={styles.iconSm} />
                    ) : (
                      <IconExpandDown className={styles.iconSm} />
                    )}
                  </button>
                )}
                <button
                  onClick={() => removeAttribute(attr.id)}
                  className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
                  title={t("stepEditor.removeAttribute")}
                >
                  <IconRemove className={styles.iconSm} />
                </button>
              </div>
              {isMappingExpanded && (
                <div className={formStyles.attrMappingPanel}>
                  {stepType === "flow" ? (
                    <select
                      value={attr.mappingName || ""}
                      onChange={(e) =>
                        updateAttribute(attr.id, "mappingName", e.target.value)
                      }
                      className={`${formStyles.formControl} ${formStyles.mappingInlineInput}`}
                      disabled={
                        attr.attrType === "output"
                          ? flowOutputList.length === 0
                          : flowInputList.length === 0
                      }
                    >
                      <option value="">
                        {t("stepEditor.flowMapPlaceholder")}
                      </option>
                      {attr.attrType === "output"
                        ? flowOutputList.map((option) => (
                            <option
                              key={option}
                              value={option}
                              disabled={
                                usedOutputMappings.has(option) &&
                                usedOutputMappings.get(option) !== attr.id
                              }
                            >
                              {option}
                            </option>
                          ))
                        : flowInputList.map((option) => (
                            <option
                              key={option.name}
                              value={option.name}
                              disabled={
                                usedInputMappings.has(option.name) &&
                                usedInputMappings.get(option.name) !== attr.id
                              }
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
                  ) : (
                    <input
                      type="text"
                      value={attr.mappingName || ""}
                      onChange={(e) =>
                        updateAttribute(attr.id, "mappingName", e.target.value)
                      }
                      placeholder={t("stepEditor.mappingSourceNamePlaceholder")}
                      className={`${formStyles.formControl} ${formStyles.mappingInlineInput}`}
                    />
                  )}
                  <div
                    className={formStyles.languageSelectorGroup}
                    aria-label={t("stepEditor.mappingLanguageLabel")}
                  >
                    {MAPPING_LANGUAGE_OPTIONS.map((option) => (
                      <button
                        key={option.value}
                        type="button"
                        onClick={(e) => {
                          updateAttribute(
                            attr.id,
                            "mappingLanguage",
                            option.value
                          );
                          e.currentTarget.blur();
                        }}
                        className={`${formStyles.languageButton} ${
                          (attr.mappingLanguage || SCRIPT_LANGUAGE_JPATH) ===
                          option.value
                            ? formStyles.languageButtonActive
                            : ""
                        }`}
                        title={t(option.labelKey)}
                      >
                        {t(option.labelKey)}
                      </button>
                    ))}
                  </div>
                  <input
                    type="text"
                    value={attr.mappingScript || ""}
                    onChange={(e) =>
                      updateAttribute(attr.id, "mappingScript", e.target.value)
                    }
                    className={`${formStyles.formControl} ${formStyles.mappingScriptInlineInput}`}
                    placeholder={t("stepEditor.mappingScriptPlaceholder")}
                  />
                </div>
              )}
              {attr.validationError && (
                <div className={formStyles.attrValidationError}>
                  {attr.validationError}
                </div>
              )}
            </div>
          );
        })}
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
    contextValue,
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
    <StepEditingContext.Provider value={extendedContextValue}>
      <div className={styles.backdrop} onClick={handleBackdropClick}>
        <div
          className={styles.content}
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
              <label className={styles.headerCheckboxLabel}>
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

                  <AttributesSection />

                  <ScriptConfigEditor
                    label={t("stepEditor.predicateLabel")}
                    value={predicate}
                    onChange={setPredicate}
                    language={predicateLanguage}
                    onLanguageChange={setPredicateLanguage}
                    languageOptions={PREDICATE_LANGUAGE_OPTIONS}
                    containerClassName={formStyles.predicateEditorContainer}
                  />

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

              {error && <div className={formStyles.errorMessage}>{error}</div>}
            </div>
          </div>

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
    </StepEditingContext.Provider>
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
