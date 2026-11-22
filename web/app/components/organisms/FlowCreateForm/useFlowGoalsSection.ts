import React from "react";
import { GoalSets, Step } from "@/app/api";
import { useT } from "@/app/i18n";
import { deriveStepGoalState, getGoalTooltip } from "@/utils/flowGoalStepState";
import {
  activeGoalSet,
  toggleGoal,
  upstreamSteps,
  withActiveGoalSet,
} from "@/utils/goalSets";
import { buildItemClassName } from "./flowFormUtils";
import styles from "./FlowGoalsSection.module.css";

export interface FlowGoalsSectionOptions {
  goalSteps: GoalSets;
  onPreviewGoalSet: (index: number | null) => void;
  onGoalStepsChange: (nextGoals: GoalSets) => void | Promise<void>;
  sortedSteps: Step[];
  satisfied: Set<string>;
  blockedByStep: Map<string, string[]>;
  missingByStep: Map<string, string[]>;
}

export interface GoalStepRow {
  step: Step;
  className: string;
  tooltipText?: string;
  isDisabled: boolean;
  handleSelect: () => void;
}

const useShiftHeld = () => {
  const [shiftHeld, setShiftHeld] = React.useState(false);

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

  return { shiftHeld, setShiftHeld };
};

export const useFlowGoalsSection = ({
  goalSteps,
  onPreviewGoalSet,
  onGoalStepsChange,
  sortedSteps,
  satisfied,
  blockedByStep,
  missingByStep,
}: FlowGoalsSectionOptions) => {
  const t = useT();
  const { shiftHeld, setShiftHeld } = useShiftHeld();
  const activeSet = activeGoalSet(goalSteps);
  const committedSets = goalSteps.slice(0, -1);
  // Disabling is scoped to the AND-set being edited, never the whole preview
  const included = React.useMemo(
    () => upstreamSteps(sortedSteps, activeSet),
    [sortedSteps, activeSet]
  );
  const hasGoals = goalSteps.some((set) => set.length > 0);
  const shouldSeedFromGoals = shiftHeld && hasGoals;
  const actionLabel = shouldSeedFromGoals
    ? t("overview.addFlowStepFromGoals")
    : t("overview.addStep");
  const actionTitle =
    shouldSeedFromGoals || !hasGoals
      ? actionLabel
      : t("overview.addStepShiftHint");

  const findStep = (id: string) => sortedSteps.find((s) => s.id === id);

  const handleRemoveSet = (idx: number) => {
    onPreviewGoalSet(null);
    void onGoalStepsChange(goalSteps.filter((_, i) => i !== idx));
  };

  const handleAddFallback = () => {
    void onGoalStepsChange([...goalSteps, []]);
    onPreviewGoalSet(null);
  };

  const stepRows: GoalStepRow[] = sortedSteps.map((step) => {
    const state = deriveStepGoalState(step.id, activeSet, {
      included,
      satisfied,
      blockedByStep,
      missingByStep,
    });
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
    return {
      step,
      className: `${itemClassName} ${includedClassName}`,
      tooltipText: getGoalTooltip(state, t),
      isDisabled: state.isDisabled,
      handleSelect,
    };
  });

  return {
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
  };
};
