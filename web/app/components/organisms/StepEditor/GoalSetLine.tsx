import React from "react";
import { IconClose } from "@/utils/iconRegistry";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import ComboInput from "@/app/components/molecules/ComboInput";
import localStyles from "./StepEditorFlowConfiguration.module.css";
import { GoalSetLineOptions, useGoalSetLine } from "./useGoalSetLine";

type TFn = (key: string, vars?: Record<string, string | number>) => string;

interface GoalSetLineProps extends GoalSetLineOptions {
  t: TFn;
}

// GoalSetLine edits one AND-set. Its suggestions exclude only the Steps
// upstream of goals on this line, never those of other OR lines
const GoalSetLine: React.FC<GoalSetLineProps> = ({ t, ...options }) => {
  const { draft, suggestionIds, goals, handleDraftChange, handleRemove } =
    useGoalSetLine(options);

  return (
    <div className={localStyles.flowGoalList} data-testid="flow-goal-line">
      {goals.map(({ id, step }) => (
        <span key={id} className={localStyles.flowGoalLabel}>
          {step && <StepTypeLabel step={step} />}
          {id}
          <button
            type="button"
            className={localStyles.flowGoalRemove}
            aria-label={t("goals.removeGoal", { id })}
            onClick={() => handleRemove(id)}
          >
            <IconClose aria-hidden="true" />
          </button>
        </span>
      ))}
      <ComboInput
        ariaLabel={t("goals.addGoal")}
        className={localStyles.flowGoalInput}
        onChange={handleDraftChange}
        placeholder={t("goals.addGoal")}
        suggestions={suggestionIds}
        value={draft}
      />
    </div>
  );
};

export default GoalSetLine;
