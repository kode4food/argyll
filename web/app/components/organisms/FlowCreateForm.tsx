import React from "react";
import { Play } from "lucide-react";
import { AttributeRole } from "../../api";
import Spinner from "../atoms/Spinner";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { useUI } from "../../contexts/UIContext";
import LazyCodeEditor from "../molecules/LazyCodeEditor";
import StepTypeLabel from "../atoms/StepTypeLabel";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreation } from "../../contexts/FlowCreationContext";

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
  const [showTopFade, setShowTopFade] = React.useState(false);
  const [showBottomFade, setShowBottomFade] = React.useState(false);
  const sidebarListRef = React.useRef<HTMLDivElement>(null);

  useEscapeKey(showCreateForm, () => setShowCreateForm(false));

  React.useEffect(() => {
    try {
      JSON.parse(initialState);
      setJsonError(null);
    } catch (error: any) {
      setJsonError(error.message);
    }
  }, [initialState]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  React.useEffect(() => {
    if (!showCreateForm) return;

    const el = sidebarListRef.current;
    if (!el) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = el;
      const hasOverflow = scrollHeight > clientHeight;

      if (!hasOverflow) {
        setShowTopFade(false);
        setShowBottomFade(false);
        return;
      }

      const atTop = scrollTop <= 1;
      const atBottom = scrollTop >= scrollHeight - clientHeight - 1;

      setShowTopFade(!atTop);
      setShowBottomFade(!atBottom);
    };

    handleScroll();

    el.addEventListener("scroll", handleScroll, { passive: true });
    window.addEventListener("resize", handleScroll);

    return () => {
      el.removeEventListener("scroll", handleScroll);
      window.removeEventListener("resize", handleScroll);
    };
  }, [showCreateForm, sortedSteps]);

  const included = React.useMemo(() => {
    if (!previewPlan?.steps) return new Set<string>();
    return new Set(Object.keys(previewPlan.steps));
  }, [previewPlan?.steps]);

  const parsedState = React.useMemo(() => {
    try {
      return JSON.parse(initialState);
    } catch {
      return {};
    }
  }, [initialState]);

  const satisfied = React.useMemo(() => {
    const satisfied = new Set<string>();
    const availableAttrs = new Set(Object.keys(parsedState));

    steps.forEach((step) => {
      const outputKeys = Object.entries(step.attributes || {})
        .filter(([_, spec]) => spec.role === AttributeRole.Output)
        .map(([name]) => name);

      if (outputKeys.length > 0) {
        const allOutputsAvailable = outputKeys.every((outputName) =>
          availableAttrs.has(outputName)
        );
        if (allOutputsAvailable) {
          satisfied.add(step.id);
        }
      }
    });

    return satisfied;
  }, [parsedState, steps]);

  if (!showCreateForm) return null;

  return (
    <>
      <div
        className={styles.overlay}
        onClick={() => setShowCreateForm(false)}
        aria-label="Close flow form"
      />
      <div className={`${styles.modal} shadow-lg`}>
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

                return (
                  <div
                    key={step.id}
                    className={`${styles.dropdownItem} ${
                      isSelected ? styles.dropdownItemSelected : ""
                    } ${isDisabled ? styles.dropdownItemDisabled : ""}`}
                    title={tooltipText}
                    onClick={async () => {
                      if (isDisabled) return;
                      const newGoalStepIds = isSelected
                        ? goalSteps.filter((id) => id !== step.id)
                        : [...goalSteps, step.id];
                      await handleStepChange(newGoalStepIds);
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
                {creating ? (
                  <Spinner size="sm" color="white" className="mr-2" />
                ) : (
                  <Play className="mr-2 icon" />
                )}
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
