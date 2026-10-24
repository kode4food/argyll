import {
  AttributeRole,
  AttributeSpec,
  ExecutionResult,
  GoalSets,
  Step,
} from "@/app/api";

// The engine's skip reason for a Goal pruned by a required match
const REQUIRED_MATCH_SKIP_REASON = "required match did not match";

const isInput = (spec: AttributeSpec): boolean =>
  spec.role === AttributeRole.Required || spec.role === AttributeRole.Optional;

const isPrunedByMatch = (exec?: ExecutionResult): boolean =>
  exec?.status === "skipped" &&
  exec.error_message === REQUIRED_MATCH_SKIP_REASON;

const isGoalSetFailed = (
  set: string[],
  execs: Map<string, ExecutionResult>
): boolean =>
  set.some((id) => execs.get(id)?.status === "failed") ||
  set.every((id) => isPrunedByMatch(execs.get(id)));

export const activeGoalSet = (goals: GoalSets): string[] =>
  goals[goals.length - 1] ?? [];

export const withActiveGoalSet = (goals: GoalSets, set: string[]): GoalSets => [
  ...goals.slice(0, -1),
  set,
];

export const nonEmptyGoalSets = (goals: GoalSets): GoalSets =>
  goals.filter((set) => set.length > 0);

export const formatGoals = (goals: GoalSets): string =>
  goals.map((set) => set.join(", ")).join(" ELSE ");

// upstreamSteps returns every Step that can feed the given goals through the
// catalog's Attribute providers. Scoped to one AND-set by its callers
export function upstreamSteps(steps: Step[], goals: string[]): Set<string> {
  const providers = new Map<string, string[]>();
  steps.forEach((step) => {
    Object.entries(step.attributes || {}).forEach(([name, spec]) => {
      if (spec.role !== AttributeRole.Output) return;
      providers.set(name, [...(providers.get(name) ?? []), step.id]);
    });
  });

  const byId = new Map(steps.map((step) => [step.id, step]));
  const res = new Set<string>();
  const todo = [...goals];
  while (todo.length > 0) {
    const step = byId.get(todo.pop()!);
    Object.entries(step?.attributes || {}).forEach(([name, spec]) => {
      if (!isInput(spec)) return;
      (providers.get(name) ?? []).forEach((id) => {
        if (res.has(id)) return;
        res.add(id);
        todo.push(id);
      });
    });
  }
  return res;
}

// currentGoalScope returns the Steps the first unfailed Goal set can use, or
// every Step once all sets have failed
export const currentGoalScope = (
  steps: Step[],
  sets: GoalSets,
  executions: ExecutionResult[]
): Set<string> => {
  const execs = new Map(executions.map((exec) => [exec.step_id, exec]));
  const current = sets.find((set) => !isGoalSetFailed(set, execs));
  if (!current) return new Set(steps.map((step) => step.id));
  return new Set([...current, ...upstreamSteps(steps, current)]);
};

// addGoal appends a goal to one AND-set, pruning goals already upstream of it
export const addGoal = (
  steps: Step[],
  set: string[],
  stepId: string
): string[] => {
  const upstream = upstreamSteps(steps, [stepId]);
  return [...set.filter((id) => !upstream.has(id)), stepId];
};

export const toggleGoal = (
  steps: Step[],
  set: string[],
  stepId: string
): string[] =>
  set.includes(stepId)
    ? set.filter((id) => id !== stepId)
    : addGoal(steps, set, stepId);
