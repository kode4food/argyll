import type { Goals } from "./types";

// GoalSets lists a goal chain's sets in the order they are attempted, the
// shape the editors work with
export type GoalSets = string[][];

export const toGoalSets = (goals?: Goals): GoalSets =>
  goals ? [goals.steps, ...toGoalSets(goals.else)] : [];

export const fromGoalSets = (sets: GoalSets): Goals =>
  sets.reduceRight<Goals | undefined>(
    (rest, steps) => (rest ? { steps, else: rest } : { steps }),
    undefined
  ) ?? { steps: [] };
