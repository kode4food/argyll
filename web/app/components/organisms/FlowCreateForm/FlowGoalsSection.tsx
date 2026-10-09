import React from "react";
import { GoalSets, Step } from "@/app/api";
import {
  getStepActionIcon,
  IconAddStep,
  IconClose,
  IconFlowGoals,
  IconGoalFallback,
  IconStepTypeFlow,
} from "@/utils/iconRegistry";
import { getStepType } from "@/utils/stepUtils";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import useArrowFocus from "@/app/hooks/useArrowFocus";
import { useT } from "@/app/i18n";
import { buildItemClassName } from "./flowFormUtils";
import { deriveStepGoalState, getGoalTooltip } from "@/utils/flowGoalStepState";
import {
  activeGoalSet,
  toggleGoal,
  upstreamSteps,
  withActiveGoalSet,
} from "@/utils/goalSets";
import styles from "./FlowGoalsSection.module.css";

interface FlowGoalsSectionProps {
  goalSteps: GoalSets;
  blockedByStep: Map<string, string[]>;
  missingByStep: Map<string, string[]>;
  onCreateStep?: (fromGoals: boolean) => void;
  onGoalStepsChange: (nextGoals: GoalSets) => void | Promise<void>;
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
  const activeSet = activeGoalSet(goalSteps);
  const committedSets = goalSteps.slice(0, -1);
  // Disabling is scoped to the AND-set being edited, never the whole preview
  const included = React.useMemo(
    () => upstreamSteps(sortedSteps, activeSet),
    [sortedSteps, activeSet]
  );
  const hasGoals = goalSteps.some((set) => set.length > 0);
  const seedsFromGoals = shiftHeld && hasGoals;
  const actionLabel = seedsFromGoals
    ? t("overview.addFlowStepFromGoals")
    : t("overview.addStep");
  const actionTitle =
    seedsFromGoals || !hasGoals ? actionLabel : t("overview.addStepShiftHint");

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
      {committedSets.map((set, idx) => (
        <div key={idx} className={styles.goalSetPanel}>
          <div className={styles.goalSetSummary} data-testid="goal-set-summary">
            {set.map((id) => {
              const step = sortedSteps.find((s) => s.id === id);
              const TypeIcon = step ? getStepActionIcon(step) : null;
              return (
                <span key={id} className={styles.goalPill}>
                  {TypeIcon && step && (
                    <TypeIcon
                      className={`step-type-icon ${getStepType(step)}`}
                    />
                  )}
                  {id}
                </span>
              );
            })}
            <button
              type="button"
              className={styles.goalSetRemove}
              aria-label={t("goals.removeSet")}
              title={t("goals.removeSet")}
              onClick={() =>
                void onGoalStepsChange(goalSteps.filter((_, i) => i !== idx))
              }
            >
              <IconClose aria-hidden="true" />
            </button>
          </div>
          <div className={styles.orDivider}>
            {t("goals.fallbackTo")}
            <IconGoalFallback size={16} aria-hidden="true" />
          </div>
        </div>
      ))}
      <div className={styles.goalListShell}>
        <div
          ref={sidebarListRef}
          onKeyDown={handleArrowFocus}
          className={`${styles.sidebarList} ${
            showTopFade ? styles.fadeTop : ""
          } ${showBottomFade ? styles.fadeBottom : ""}`}
        >
          {sortedSteps.map((step) => {
            const state = deriveStepGoalState(step.id, activeSet, {
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
              void onGoalStepsChange(
                withActiveGoalSet(
                  goalSteps,
                  toggleGoal(sortedSteps, activeSet, step.id)
                )
              );
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
        <button
          type="button"
          className={styles.orButton}
          disabled={activeSet.length === 0}
          aria-label={t("goals.addFallback")}
          title={t("goals.addFallback")}
          onClick={() => void onGoalStepsChange([...goalSteps, []])}
        >
          <IconGoalFallback size={16} aria-hidden="true" />
        </button>
      </div>
    </section>
  );
};

export default FlowGoalsSection;
