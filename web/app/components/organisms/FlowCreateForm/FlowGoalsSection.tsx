import React from "react";
import { Step } from "@/app/api";
import { IconAddStep } from "@/utils/iconRegistry";
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
  onCreateStep?: () => void;
  onGoalStepsChange: (nextGoalStepIds: string[]) => void | Promise<void>;
  satisfied: Set<string>;
  showBottomFade: boolean;
  showTopFade: boolean;
  sidebarListRef: React.RefObject<HTMLDivElement | null>;
  sortedSteps: Step[];
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
  stepsCount,
}) => {
  const t = useT();
  const handleArrowFocus = useArrowFocus();

  return (
    <section className={`${styles.sectionCard} ${styles.stepSection}`}>
      <div className={styles.sectionHeader}>
        <div className={styles.sectionTitle}>
          {t("stepEditor.flowGoalsLabel")}
        </div>
        <div className={styles.sectionHeaderActions}>
          <div className={styles.sectionMeta}>
            {t("overview.stepsRegistered", {
              count: stepsCount,
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
