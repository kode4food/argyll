import { ExecutionResult, WorkState } from "@/app/api";

const addUsage = (
  current: Record<string, number> | undefined,
  usage: Record<string, number>
): Record<string, number> => {
  const total: Record<string, number> = Object.assign(
    Object.create(null),
    current
  );
  for (const [name, value] of Object.entries(usage)) {
    total[name] = (total[name] ?? 0) + value;
  }
  return total;
};

export const mergeWorkItem = (
  execution: ExecutionResult,
  token: string,
  { usage, ...updates }: Partial<WorkState>
): ExecutionResult => {
  const workItems = execution.work_items || {};
  const existingItem = workItems[token] || {
    token,
    status: "pending",
    inputs: {},
    retry_count: 0,
  };
  const work = { ...existingItem, ...updates };
  if (usage) {
    work.usage = addUsage(work.usage, usage);
  }

  return {
    ...execution,
    work_items: {
      ...workItems,
      [token]: work,
    },
  };
};
