import React from "react";
import { IconStartFlow } from "@/utils/iconRegistry";
import Spinner from "@/app/components/atoms/Spinner";
import { useEscapeKey } from "@/app/hooks/useEscapeKey";
import { useUI } from "@/app/contexts/UIContext";
import LazyCodeEditor from "@/app/components/molecules/LazyCodeEditor";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreation } from "@/app/contexts/FlowCreationContext";
import { useFlowFormScrollFade } from "./useFlowFormScrollFade";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";
import {
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  buildItemClassName,
  validateJsonString,
} from "./flowFormUtils";
import { useT } from "@/app/i18n";
import { getFlowPlanAttributeOptions } from "@/utils/flowPlanAttributeOptions";

const FlowCreateForm: React.FC = () => {
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
  const { showCreateForm, setShowCreateForm, previewPlan, goalSteps } = useUI();
  const t = useT();

  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");

  useEscapeKey(showCreateForm, () => setShowCreateForm(false));

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  React.useEffect(() => {
    if (showCreateForm) {
      setEditorMode("basic");
    }
  }, [showCreateForm]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  const { sidebarListRef, showTopFade, showBottomFade } =
    useFlowFormScrollFade(showCreateForm);

  const { included, satisfied, missingByStep } = useFlowFormStepFiltering(
    steps,
    initialState,
    previewPlan
  );
  const { flowInputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(previewPlan),
    [previewPlan]
  );
  const flowInputNames = React.useMemo(
    () => flowInputOptions.map((option) => option.name),
    [flowInputOptions]
  );
  const flowInputValues = React.useMemo(
    () => buildInitialStateInputValues(initialState, flowInputNames),
    [flowInputNames, initialState]
  );

  const handleBasicInputChange = React.useCallback(
    (name: string, value: string) => {
      const nextValues = {
        ...flowInputValues,
        [name]: value,
      };
      const nextState = buildInitialStateFromInputValues(
        nextValues,
        flowInputNames
      );
      setInitialState(JSON.stringify(nextState, null, 2));
    },
    [flowInputNames, flowInputValues, setInitialState]
  );

  if (!showCreateForm) return null;

  return (
    <>
      <div
        className={styles.overlay}
        onClick={() => setShowCreateForm(false)}
        aria-label={t("flowCreate.closeForm")}
      />
      <div className={styles.modal}>
        <div className={styles.container}>
          <div className={styles.sidebar}>
            <div className={styles.sidebarHeader}>
              <label className={styles.label}>
                {t("flowCreate.selectGoalSteps")}
              </label>
            </div>
            <div
              ref={sidebarListRef}
              className={`${styles.sidebarList} ${
                showTopFade ? styles.fadeTop : ""
              } ${showBottomFade ? styles.fadeBottom : ""}`}
            >
              {sortedSteps.map((step) => {
                const isSelected = goalSteps.includes(step.id);
                const isIncludedByOthers = included.has(step.id) && !isSelected;
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
                    onClick={async () => {
                      if (isDisabled) return;
                      const newGoalStepIds = isSelected
                        ? goalSteps.filter((id) => id !== step.id)
                        : [...goalSteps, step.id];
                      handleStepChange(newGoalStepIds);
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

          <div className={styles.main}>
            <div>
              <label className={styles.label}>
                {t("flowCreate.flowIdLabel")}
              </label>
              <div className={styles.idGroup}>
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
                  className={styles.buttonGenerate}
                  title={t("flowCreate.generateIdTitle")}
                  aria-label={t("flowCreate.generateIdAria")}
                >
                  ↻
                </button>
              </div>
            </div>

            <div className={styles.editorContainer}>
              {editorMode === "json" && (
                <div
                  className={`${styles.editorHeader} ${styles.editorHeaderWithLabel}`}
                >
                  <span className={styles.editorModeLabel}>
                    {t("flowCreate.requiredAttributesLabel")}
                  </span>
                </div>
              )}
              {editorMode === "basic" ? (
                <div className={styles.editorWrapper}>
                  <div className={styles.attributeTableScroll}>
                    <div className={styles.attributeList}>
                      <div className={styles.attributeListHeader}>
                        <div className={styles.attributeListHeaderCell}>
                          {t("flowCreate.attributeColumn")}
                        </div>
                        <div className={styles.attributeListHeaderCell}>
                          {t("flowCreate.valueColumn")}
                        </div>
                      </div>

                      {flowInputOptions.map((option) => (
                        <div
                          key={option.name}
                          className={styles.attributeListItem}
                        >
                          <div className={styles.attributeNameCell}>
                            <span className={styles.attributeNameText}>
                              {option.name}
                            </span>
                            <span className={styles.requiredBadgeSlot}>
                              {option.required && (
                                <span className={styles.requiredBadge}>
                                  {t("flowCreate.requiredBadge")}
                                </span>
                              )}
                            </span>
                          </div>
                          <div className={styles.attributeValueCell}>
                            <input
                              type="text"
                              value={flowInputValues[option.name] || ""}
                              onChange={(e) =>
                                handleBasicInputChange(
                                  option.name,
                                  e.target.value
                                )
                              }
                              className={`${styles.input} ${styles.attributeValueInput}`}
                              placeholder={t("flowCreate.valuePlaceholder")}
                            />
                          </div>
                        </div>
                      ))}

                      {flowInputOptions.length === 0 && (
                        <div className={styles.emptyAttributes}>
                          {t("flowCreate.noPotentialInputs")}
                        </div>
                      )}
                    </div>
                  </div>
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

            <div className={styles.actions}>
              <div className={styles.actionsLeft}>
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
              <div className={styles.actionsRight}>
                <button
                  onClick={() => setShowCreateForm(false)}
                  className={styles.buttonCancel}
                >
                  {t("common.cancel")}
                </button>
                <button
                  onClick={handleCreateFlow}
                  disabled={
                    creating ||
                    !newID.trim() ||
                    goalSteps.length === 0 ||
                    (editorMode === "json" && jsonError !== null)
                  }
                  className={styles.buttonStart}
                >
                  <span className={styles.buttonIcon}>
                    {creating ? (
                      <Spinner size="sm" color="white" />
                    ) : (
                      <IconStartFlow className={styles.startIcon} />
                    )}
                  </span>
                  {t("common.start")}
                </button>
              </div>
            </div>
          </div>
        </div>
        {steps.length === 0 && (
          <div className={styles.warning}>{t("flowCreate.warningNoSteps")}</div>
        )}
      </div>
    </>
  );
};

export default FlowCreateForm;
