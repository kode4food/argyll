import React from "react";
import { GoalSets, Step } from "@/app/api";
import {
  IconAddStep,
  IconClose,
  IconFlowGoals,
  IconGoalFallback,
  IconStepTypeFlow,
} from "@/utils/iconRegistry";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import useArrowFocus from "@/app/hooks/useArrowFocus";
import { useT } from "@/app/i18n";
import { useLocale } from "@/app/store/i18nStore";
import styles from "./FlowGoalsSection.module.css";
import { useFlowGoalsSection } from "./useFlowGoalsSection";
import fadeStyles from "@/app/styles/components/scrollFade.module.css";

interface FlowGoalsSectionProps {
  goalSteps: GoalSets;
  onPreviewGoalSet: (index: number | null) => void;
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
  onPreviewGoalSet,
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
  const locale = useLocale();
  const handleArrowFocus = useArrowFocus();
  const {
    setShiftHeld,
    activeSet,
    committedSets,
    shouldSeedFromGoals,
    actionLabel,
    actionTitle,
    findStep,
    handleRemoveSet,
    handleAddFallback,
    stepRows,
  } = useFlowGoalsSection({
    goalSteps,
    onPreviewGoalSet,
    onGoalStepsChange,
    sortedSteps,
    satisfied,
    blockedByStep,
    missingByStep,
  });

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
              {shouldSeedFromGoals ? (
                <IconStepTypeFlow className={styles.sectionActionIcon} />
              ) : (
                <IconAddStep className={styles.sectionActionIcon} />
              )}
            </button>
          )}
        </div>
      </div>
      {committedSets.length > 0 && (
        <div className={styles.goalSetPanel}>
          {committedSets.map((set, idx) => (
            <div
              key={idx}
              className={styles.goalSetSummary}
              data-testid="goal-set-summary"
              tabIndex={0}
              onPointerEnter={() => onPreviewGoalSet(idx)}
              onPointerLeave={() => onPreviewGoalSet(null)}
              onFocus={() => onPreviewGoalSet(idx)}
              onBlur={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget))
                  onPreviewGoalSet(null);
              }}
            >
              <span className={styles.orDivider} lang={locale}>
                <span className={styles.orLabel}>{t("goals.or")}</span>
              </span>
              <span className={styles.goalSetContent}>
                {set.map((id) => {
                  const step = findStep(id);
                  return (
                    <span key={id} className={styles.goalPill}>
                      {step && <StepTypeLabel step={step} />}
                      {id}
                    </span>
                  );
                })}
              </span>
              <button
                type="button"
                className={styles.goalSetRemove}
                aria-label={t("goals.removeSet")}
                title={t("goals.removeSet")}
                onClick={() => handleRemoveSet(idx)}
              >
                <IconClose aria-hidden="true" />
              </button>
            </div>
          ))}
        </div>
      )}
      <div className={styles.goalListShell}>
        <div
          ref={sidebarListRef}
          onKeyDown={handleArrowFocus}
          className={`${styles.sidebarList} ${
            showTopFade ? fadeStyles.fadeTop : ""
          } ${showBottomFade ? fadeStyles.fadeBottom : ""}`}
        >
          {stepRows.length === 0 && (
            <div className={styles.emptySteps}>
              {t("flowCreate.noSteps")
                .split("{button}")
                .flatMap((part, i) => [
                  i > 0 && (
                    <span
                      key={`button-${i}`}
                      className={styles.emptyStepsAction}
                      role="img"
                      aria-label={t("overview.addStep")}
                    >
                      <IconAddStep className={styles.sectionActionIcon} />
                    </span>
                  ),
                  part && <span key={`text-${i}`}>{part}</span>,
                ])}
            </div>
          )}
          {stepRows.map(
            ({ step, className, tooltipText, isDisabled, handleSelect }) => (
              <div
                key={step.id}
                className={className}
                title={tooltipText}
                role="button"
                aria-disabled={isDisabled}
                data-arrow-focus-item="true"
                tabIndex={isDisabled ? -1 : 0}
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
            )
          )}
        </div>
        <button
          type="button"
          className={styles.orButton}
          disabled={activeSet.length === 0}
          aria-label={t("goals.addFallback")}
          title={t("goals.addFallback")}
          onClick={handleAddFallback}
        >
          <IconGoalFallback size={16} aria-hidden="true" />
        </button>
      </div>
    </section>
  );
};

export default FlowGoalsSection;
