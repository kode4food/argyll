import React from "react";
import { IconAddStep, IconStartFlow } from "@/utils/iconRegistry";
import Spinner from "@/app/components/atoms/Spinner";
import { useUI } from "@/app/contexts/UIContext";
import LazyCodeEditor from "@/app/components/molecules/LazyCodeEditor";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import { AttributeType } from "@/app/api";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreation } from "@/app/contexts/FlowCreationContext";
import { useFlowFormScrollFade } from "./useFlowFormScrollFade";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";
import {
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  buildItemClassName,
  FlowInputStatus,
  getFlowInputStatus,
  validateJsonString,
} from "./flowFormUtils";
import { useT } from "@/app/i18n";
import {
  FlowInputOption,
  getFlowPlanAttributeOptions,
} from "@/utils/flowPlanAttributeOptions";

interface FlowCreateFormProps {
  onCreateStep?: () => void;
}

const getTypePlaceholder = (type?: AttributeType): string => {
  switch (type) {
    case AttributeType.Number:
      return "0";
    case AttributeType.Boolean:
      return "false";
    case AttributeType.Object:
      return "{}";
    case AttributeType.Array:
      return "[]";
    case AttributeType.String:
      return '""';
    case AttributeType.Null:
      return "null";
    default:
      return "";
  }
};

const getFlowInputPlaceholder = (option: FlowInputOption): string => {
  if (option.defaultValue !== undefined) {
    return option.defaultValue;
  }

  return getTypePlaceholder(option.type);
};

const FlowCreateForm: React.FC<FlowCreateFormProps> = ({ onCreateStep }) => {
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
    generateID,
    sortSteps,
  } = useFlowCreation();
  const {
    previewPlan,
    goalSteps,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  } = useUI();
  const t = useT();

  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  React.useEffect(() => {
    if (editorMode !== "basic") {
      setFocusedPreviewAttribute(null);
    }
  }, [editorMode, setFocusedPreviewAttribute]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  const { sidebarListRef, showTopFade, showBottomFade } =
    useFlowFormScrollFade(true);

  const { included, satisfied, missingByStep } = useFlowFormStepFiltering(
    steps,
    initialState,
    previewPlan
  );
  const { flowInputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(previewPlan),
    [previewPlan]
  );
  const emptyAttributesLabel =
    goalSteps.length === 0
      ? t("flowCreate.noGoalsSelected")
      : t("flowCreate.noPotentialInputs");
  const flowInputNames = React.useMemo(
    () => flowInputOptions.map((option) => option.name),
    [flowInputOptions]
  );

  React.useEffect(() => {
    if (editorMode !== "basic") {
      return;
    }

    if (
      focusedPreviewAttribute &&
      !flowInputNames.includes(focusedPreviewAttribute)
    ) {
      setFocusedPreviewAttribute(null);
    }
  }, [
    editorMode,
    flowInputNames,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  ]);
  const flowInputValuesRaw = React.useMemo(
    () => buildInitialStateInputValues(initialState, flowInputNames),
    [flowInputNames, initialState]
  );
  const flowInputValues = React.useMemo(() => {
    const values: Record<string, string> = {};

    flowInputOptions.forEach((option) => {
      const rawValue = flowInputValuesRaw[option.name] || "";
      const status = getFlowInputStatus(option, rawValue);
      values[option.name] = status === "defaulted" ? "" : rawValue;
    });

    return values;
  }, [flowInputOptions, flowInputValuesRaw]);

  const statusLabelByType: Record<FlowInputStatus, string> = {
    provided: t("flowCreate.providedBadge"),
    defaulted: t("flowCreate.defaultBadge"),
    required: t("flowCreate.requiredBadge"),
    optional: t("flowStats.optionalLabel"),
  };
  const statusClassByType: Record<FlowInputStatus, string> = {
    provided: styles.requiredBadgeSatisfied,
    defaulted: styles.requiredBadgeDefault,
    required: styles.requiredBadgeMissing,
    optional: styles.requiredBadgeOptional,
  };

  const handleBasicInputChange = React.useCallback(
    (name: string, value: string) => {
      const option = flowInputOptions.find((item) => item.name === name);
      const normalizedValue =
        value.trim() === "" &&
        option?.required &&
        option.defaultValue !== undefined
          ? option.defaultValue
          : value;
      const nextValues = {
        ...flowInputValuesRaw,
        [name]: normalizedValue,
      };
      const nextState = buildInitialStateFromInputValues(
        nextValues,
        flowInputNames
      );

      setInitialState(JSON.stringify(nextState, null, 2));
    },
    [flowInputNames, flowInputOptions, flowInputValuesRaw, setInitialState]
  );

  return (
    <div className={styles.panel}>
      <div className={styles.container}>
        <div className={styles.main}>
          <div className={styles.panelBody}>
            <section className={`${styles.sectionCard} ${styles.stepSection}`}>
              <div className={styles.sectionHeader}>
                <div className={styles.sectionTitle}>
                  {t("stepEditor.flowGoalsLabel")}
                </div>
                <div className={styles.sectionHeaderActions}>
                  <div className={styles.sectionMeta}>
                    {t("overview.stepsRegistered", {
                      count: steps.length,
                    })}
                  </div>
                  {onCreateStep && (
                    <button
                      type="button"
                      className={styles.sectionActionButton}
                      title={t("overview.addStep")}
                      aria-label={t("overview.addStep")}
                      onClick={onCreateStep}
                    >
                      <IconAddStep className={styles.sectionActionIcon} />
                    </button>
                  )}
                </div>
              </div>
              <div className={styles.goalListShell}>
                <div
                  ref={sidebarListRef}
                  className={`${styles.sidebarList} ${
                    showTopFade ? styles.fadeTop : ""
                  } ${showBottomFade ? styles.fadeBottom : ""}`}
                >
                  {sortedSteps.map((step) => {
                    const isSelected = goalSteps.includes(step.id);
                    const isIncludedByOthers =
                      included.has(step.id) && !isSelected;
                    const isSatisfiedByState =
                      satisfied.has(step.id) && !isSelected;
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
                    const itemClassName = buildItemClassName(
                      isSelected,
                      isDisabled,
                      styles.dropdownItem,
                      styles.dropdownItemSelected,
                      styles.dropdownItemDisabled
                    );
                    const includedClassName = isIncludedByOthers
                      ? styles.dropdownItemIncluded
                      : "";

                    return (
                      <div
                        key={step.id}
                        className={`${itemClassName} ${includedClassName}`}
                        title={tooltipText}
                        onClick={() => {
                          if (isDisabled) {
                            return;
                          }

                          const nextGoalStepIds = isSelected
                            ? goalSteps.filter((id) => id !== step.id)
                            : [...goalSteps, step.id];

                          void handleStepChange(nextGoalStepIds);
                        }}
                      >
                        <table className={styles.stepTable}>
                          <tbody>
                            <tr>
                              <td className={styles.stepCellType}>
                                <StepTypeLabel step={step} />
                              </td>
                              <td className={styles.stepCellName}>
                                <div>{step.name}</div>
                                <div className={styles.stepId}>({step.id})</div>
                              </td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    );
                  })}
                </div>
              </div>
            </section>

            <section
              className={`${styles.sectionCard} ${styles.attributesSection}`}
            >
              <div className={styles.sectionHeader}>
                <div className={styles.sectionTitle}>
                  {t("flowCreate.requiredAttributesLabel")}
                </div>
                <div className={styles.editorModeToggleGroup}>
                  <button
                    type="button"
                    className={`${styles.editorModeToggle} ${
                      editorMode === "basic"
                        ? styles.editorModeToggleActive
                        : ""
                    }`}
                    onClick={() => setEditorMode("basic")}
                  >
                    {t("flowCreate.modeBasic")}
                  </button>
                  <button
                    type="button"
                    className={`${styles.editorModeToggle} ${
                      editorMode === "json" ? styles.editorModeToggleActive : ""
                    }`}
                    onClick={() => setEditorMode("json")}
                  >
                    {t("flowCreate.modeJson")}
                  </button>
                </div>
              </div>
              <div className={styles.editorContainer}>
                {editorMode === "basic" ? (
                  <div className={styles.editorWrapper}>
                    {flowInputOptions.length === 0 ? (
                      <div className={styles.emptyAttributesCentered}>
                        {emptyAttributesLabel}
                      </div>
                    ) : (
                      <div className={styles.attributeTableScroll}>
                        <div className={styles.attributeList}>
                          <div className={styles.attributeListHeader}>
                            <div
                              className={`${styles.attributeListHeaderCell} ${styles.attributeHeaderCell}`}
                            >
                              {t("flowCreate.attributeColumn")}
                            </div>
                            <div
                              className={`${styles.attributeListHeaderCell} ${styles.attributeValueHeaderCell}`}
                            >
                              {t("flowCreate.valueColumn")}
                            </div>
                          </div>

                          {flowInputOptions.map((option) => {
                            const value = flowInputValues[option.name] || "";
                            const rawValue =
                              flowInputValuesRaw[option.name] || "";
                            const status = getFlowInputStatus(option, rawValue);
                            const statusClass = statusClassByType[status];
                            const statusLabel = statusLabelByType[status];

                            return (
                              <div
                                key={option.name}
                                className={styles.attributeListItem}
                              >
                                <div className={styles.requiredBadgeCell}>
                                  <span
                                    className={`${styles.requiredBadge} ${statusClass}`}
                                    role="img"
                                    aria-label={statusLabel}
                                    title={statusLabel}
                                  />
                                </div>
                                <div className={styles.attributeNameCell}>
                                  <span className={styles.attributeNameText}>
                                    {option.name}
                                  </span>
                                </div>
                                <div className={styles.attributeValueCell}>
                                  <input
                                    type="text"
                                    value={value}
                                    onChange={(e) =>
                                      handleBasicInputChange(
                                        option.name,
                                        e.target.value
                                      )
                                    }
                                    onFocus={() =>
                                      setFocusedPreviewAttribute(option.name)
                                    }
                                    className={`${styles.input} ${styles.attributeValueInput}`}
                                    placeholder={getFlowInputPlaceholder(
                                      option
                                    )}
                                  />
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )}
                  </div>
                ) : (
                  <>
                    <div className={styles.editorWrapper}>
                      <LazyCodeEditor
                        value={initialState}
                        onChange={setInitialState}
                        height="100%"
                      />
                    </div>
                    {jsonError && (
                      <div className={styles.jsonError}>
                        {t("flowCreate.invalidJson", { error: jsonError })}
                      </div>
                    )}
                  </>
                )}
              </div>
            </section>
          </div>

          <div className={styles.panelFooter}>
            <div className={styles.section}>
              <label className={styles.label}>
                {t("flowCreate.startFlowLabel")}
              </label>
              <div className={styles.footerRow}>
                <div className={styles.idControls}>
                  <input
                    type="text"
                    value={newID}
                    onChange={(e) => {
                      setNewID(e.target.value);
                      setIDManuallyEdited(true);
                    }}
                    placeholder={t("flowCreate.flowIdPlaceholder")}
                    className={`${styles.input} ${styles.idInputFlex}`}
                  />
                  <button
                    type="button"
                    onClick={() => {
                      setNewID(generateID());
                      setIDManuallyEdited(false);
                    }}
                    className={`${styles.buttonGenerate} ${styles.footerIconButton}`}
                    title={t("flowCreate.generateIdTitle")}
                    aria-label={t("flowCreate.generateIdAria")}
                  >
                    ↻
                  </button>
                </div>
                <button
                  onClick={handleCreateFlow}
                  disabled={
                    creating ||
                    !newID.trim() ||
                    goalSteps.length === 0 ||
                    (editorMode === "json" && jsonError !== null)
                  }
                  className={`${styles.buttonStart} ${styles.footerIconButton}`}
                  title={t("common.start")}
                  aria-label={t("common.start")}
                >
                  {creating ? (
                    <Spinner size="sm" color="white" />
                  ) : (
                    <IconStartFlow className={styles.startIcon} />
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {steps.length === 0 && (
        <div className={styles.warning}>{t("flowCreate.warningNoSteps")}</div>
      )}
    </div>
  );
};

export default FlowCreateForm;
