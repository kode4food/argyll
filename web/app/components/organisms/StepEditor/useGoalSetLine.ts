import React from "react";
import { Step } from "@/app/api";
import {
  deriveStepGoalState,
  GoalStepContext,
} from "@/utils/flowGoalStepState";
import { addGoal, upstreamSteps } from "@/utils/goalSets";

export interface GoalSetLineOptions {
  context: Omit<GoalStepContext, "included">;
  onChange: (set: string[]) => void;
  set: string[];
  steps: Step[];
}

export const useGoalSetLine = ({
  context,
  onChange,
  set,
  steps,
}: GoalSetLineOptions) => {
  const [draft, setDraft] = React.useState("");
  const suggestions = React.useMemo(() => {
    const included = upstreamSteps(steps, set);
    return steps.filter(
      (step) =>
        !set.includes(step.id) &&
        !deriveStepGoalState(step.id, set, { ...context, included }).isDisabled
    );
  }, [context, set, steps]);

  const handleDraftChange = (value: string) => {
    if (suggestions.some((step) => step.id === value)) {
      setDraft("");
      onChange(addGoal(steps, set, value));
      return;
    }
    setDraft(value);
  };

  const goals = set.map((id) => ({
    id,
    step: steps.find((s) => s.id === id),
  }));
  const handleRemove = (id: string) =>
    onChange(set.filter((goal) => goal !== id));

  return {
    draft,
    suggestionIds: suggestions.map((step) => step.id),
    goals,
    handleDraftChange,
    handleRemove,
  };
};
