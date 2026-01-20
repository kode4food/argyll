import React from "react";
import { Play } from "lucide-react";
import Spinner from "@/app/components/atoms/Spinner";
import { useEscapeKey } from "@/app/hooks/useEscapeKey";
import { useUI } from "@/app/contexts/UIContext";
import LazyCodeEditor from "@/app/components/molecules/LazyCodeEditor";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreation } from "@/app/contexts/FlowCreationContext";
import { useFlowFormScrollFade } from "./useFlowFormScrollFade";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";
import { validateJsonString, buildItemClassName } from "./flowFormUtils";
import { useT } from "@/app/i18n";

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

  useEscapeKey(showCreateForm, () => setShowCreateForm(false));

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  const { sidebarListRef, showTopFade, showBottomFade } =
    useFlowFormScrollFade(showCreateForm);

  const { included, satisfied, missingByStep } = useFlowFormStepFiltering(
    steps,
    initialState,
    previewPlan
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
                            <div className={styles.stepId}>
                              ({step.id})
                              {isIncludedByOthers && (
                                <span className={styles.includedCheck}>✓</span>
                              )}
                            </div>
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
              <label className={styles.label}>
                {t("flowCreate.requiredAttributesLabel")}
              </label>
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
            </div>

            <div className={styles.actions}>
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
                  jsonError !== null
                }
                className={styles.buttonStart}
              >
                <span className={styles.buttonIcon}>
                  {creating ? (
                    <Spinner size="sm" color="white" />
                  ) : (
                    <Play className={styles.startIcon} />
                  )}
                </span>
                {t("common.start")}
              </button>
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
