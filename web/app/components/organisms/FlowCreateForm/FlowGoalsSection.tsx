import React from "react";
import { Step } from "@/app/api";
import {
  IconAddStep,
  IconFlowGoals,
  IconStepTypeFlow,
} from "@/utils/iconRegistry";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import useArrowFocus from "@/app/hooks/useArrowFocus";
import { useT } from "@/app/i18n";
import { buildItemClassName } from "./flowFormUtils";
import { deriveStepGoalState, getGoalTooltip } from "@/utils/flowGoalStepState";
import styles from "./FlowGoalsSection.module.css";

interface FlowGoalsSectionProps {
  goalSteps: string[];
  blockedByStep: Map<string, string[]>;
  included: Set<string>;
  missingByStep: Map<string, string[]>;
  onCreateStep?: (fromGoals: boolean) => void;
  onGoalStepsChange: (nextGoalStepIds: string[]) => void | Promise<void>;
  satisfied: Set<string>;
  showBottomFade: boolean;
  showTopFade: boolean;
  sidebarListRef: React.RefObject<HTMLDivElement | null>;
  sortedSteps: Step[];
  spaceScoped: boolean;
  stepsCount: number;
}

const FlowGoalsSection: React.FC<FlowGoalsSectionProps> = ({
  goalSteps,
  blockedByStep,
  included,
  missingByStep,
  onCreateStep,
  onGoalStepsChange,
  satisfied,
  showBottomFade,
  showTopFade,
  sidebarListRef,
  sortedSteps,
  spaceScoped,
  stepsCount,
}) => {
  const t = useT();
  const handleArrowFocus = useArrowFocus();
  const [shiftHeld, setShiftHeld] = React.useState(false);
  const seedsFromGoals = shiftHeld && goalSteps.length > 0;
  const actionLabel = seedsFromGoals
    ? t("overview.addFlowStepFromGoals")
    : t("overview.addStep");
  const actionTitle =
    seedsFromGoals || goalSteps.length === 0
      ? actionLabel
      : t("overview.addStepShiftHint");

  React.useEffect(() => {
    const track = (e: KeyboardEvent) => setShiftHeld(e.shiftKey);
    const clear = () => setShiftHeld(false);
    window.addEventListener("keydown", track);
    window.addEventListener("keyup", track);
    window.addEventListener("blur", clear);
    return () => {
      window.removeEventListener("keydown", track);
      window.removeEventListener("keyup", track);
      window.removeEventListener("blur", clear);
    };
  }, []);

  return (
    <section className={`${styles.sectionCard} ${styles.stepSection}`}>
      <div className={styles.sectionHeader}>
        <div className={styles.sectionTitle}>
          <span className={styles.sectionTitleIcon}>
            <IconFlowGoals aria-hidden="true" />
          </span>
          {t("stepEditor.flowGoalsLabel")}
        </div>
        <div className={styles.sectionHeaderActions}>
          <div className={styles.sectionMeta}>
            {t(
              spaceScoped
                ? "spaceManager.matchingSteps"
                : "overview.stepsRegistered",
              { count: stepsCount }
            )}
          </div>
          {onCreateStep && (
            <button
              type="button"
              className={styles.sectionActionButton}
              title={actionTitle}
              aria-label={actionLabel}
              onPointerEnter={(e) => setShiftHeld(e.shiftKey)}
              onClick={(e) => onCreateStep(e.shiftKey)}
            >
              {seedsFromGoals ? (
                <IconStepTypeFlow className={styles.sectionActionIcon} />
              ) : (
                <IconAddStep className={styles.sectionActionIcon} />
              )}
            </button>
          )}
        </div>
      </div>
      <div className={styles.goalListShell}>
        <div
          ref={sidebarListRef}
          onKeyDown={handleArrowFocus}
          className={`${styles.sidebarList} ${
            showTopFade ? styles.fadeTop : ""
          } ${showBottomFade ? styles.fadeBottom : ""}`}
        >
          {sortedSteps.map((step) => {
            const state = deriveStepGoalState(step.id, goalSteps, {
              included,
              satisfied,
              blockedByStep,
              missingByStep,
            });
            const tooltipText = getGoalTooltip(state, t);
            const itemClassName = buildItemClassName(
              state.isSelected,
              state.isDisabled,
              {
                base: styles.dropdownItem,
                selected: styles.dropdownItemSelected,
                disabled: styles.dropdownItemDisabled,
              }
            );
            const includedClassName = state.isIncludedByOthers
              ? styles.dropdownItemIncluded
              : "";
            const handleSelect = () => {
              if (state.isDisabled) return;
              const nextGoalStepIds = state.isSelected
                ? goalSteps.filter((id) => id !== step.id)
                : [...goalSteps, step.id];
              void onGoalStepsChange(nextGoalStepIds);
            };

            return (
              <div
                key={step.id}
                className={`${itemClassName} ${includedClassName}`}
                title={tooltipText}
                role="button"
                aria-disabled={state.isDisabled}
                data-arrow-focus-item="true"
                tabIndex={state.isDisabled ? -1 : 0}
                onClick={handleSelect}
                onKeyDown={(e) => {
                  if (e.key !== "Enter" && e.key !== " ") return;
                  e.preventDefault();
                  handleSelect();
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
  );
};

export default FlowGoalsSection;
