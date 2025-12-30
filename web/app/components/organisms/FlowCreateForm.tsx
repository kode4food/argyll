import React from "react";
import { Play } from "lucide-react";
import Spinner from "../atoms/Spinner";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { useUI } from "../../contexts/UIContext";
import LazyCodeEditor from "../molecules/LazyCodeEditor";
import StepTypeLabel from "../atoms/StepTypeLabel";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreation } from "../../contexts/FlowCreationContext";
import { useFlowFormScrollFade } from "../molecules/FlowCreateForm/useFlowFormScrollFade";
import { useFlowFormStepFiltering } from "../molecules/FlowCreateForm/useFlowFormStepFiltering";
import {
  validateJsonString,
  buildItemClassName,
} from "../molecules/FlowCreateForm/flowFormUtils";

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

  const [jsonError, setJsonError] = React.useState<string | null>(null);

  useEscapeKey(showCreateForm, () => setShowCreateForm(false));

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  const { sidebarListRef, showTopFade, showBottomFade } =
    useFlowFormScrollFade(showCreateForm);

  const { included, satisfied } = useFlowFormStepFiltering(
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
        aria-label="Close flow form"
      />
      <div className={styles.modal}>
        <div className={styles.container}>
          <div className={styles.sidebar}>
            <div className={styles.sidebarHeader}>
              <label className={styles.label}>Select Goal Steps</label>
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
                const isDisabled = isIncludedByOthers || isSatisfiedByState;

                const tooltipText = isIncludedByOthers
                  ? "Already included in execution plan"
                  : isSatisfiedByState
                    ? "Outputs satisfied by initial state"
                    : undefined;
                const itemClassName = buildItemClassName(
                  isSelected,
                  isDisabled,
                  styles.dropdownItem,
                  styles.dropdownItemSelected,
                  styles.dropdownItemDisabled
                );

                return (
                  <div
                    key={step.id}
                    className={itemClassName}
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
              <label className={styles.label}>Flow ID</label>
              <div className={styles.idGroup}>
                <input
                  type="text"
                  value={newID}
                  onChange={(e) => {
                    setNewID(e.target.value);
                    setIDManuallyEdited(true);
                  }}
                  placeholder="e.g., order-processing-001"
                  className={`${styles.input} ${styles.idInputFlex}`}
                />
                <button
                  type="button"
                  onClick={() => {
                    setNewID(generateID());
                    setIDManuallyEdited(false);
                  }}
                  className={styles.buttonGenerate}
                  title="Generate new ID"
                  aria-label="Generate new flow ID"
                >
                  ↻
                </button>
              </div>
            </div>

            <div className={styles.editorContainer}>
              <label className={styles.label}>Required Attributes</label>
              <div className={styles.editorWrapper}>
                <LazyCodeEditor
                  value={initialState}
                  onChange={setInitialState}
                  height="100%"
                />
              </div>
              {jsonError && (
                <div className={styles.jsonError}>
                  Invalid JSON: {jsonError}
                </div>
              )}
            </div>

            <div className={styles.actions}>
              <button
                onClick={() => setShowCreateForm(false)}
                className={styles.buttonCancel}
              >
                Cancel
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
                Start
              </button>
            </div>
          </div>
        </div>
        {steps.length === 0 && (
          <div className={styles.warning}>
            ⚠️ No steps are registered. Flows need registered steps to function.
          </div>
        )}
      </div>
    </>
  );
};

export default FlowCreateForm;
