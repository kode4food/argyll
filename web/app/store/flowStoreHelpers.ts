import {
  AttributeValue,
  ExecutionPlan,
  ExecutionResult,
  FlowContext,
  FlowSummary,
  QueryFlowsItem,
  Step,
} from "../api";

export interface FlowStateUpdate {
  id: string;
  status: FlowContext["status"];
  attributes?: Record<string, any>;
  plan?: ExecutionPlan;
  executions?: Record<string, any>;
  created_at?: string;
  completed_at?: string;
  error?: string;
}

export const isRunningFlow = (status: FlowSummary["status"]): boolean => {
  return status === "pending" || status === "active";
};

export const compareFlows = (a: FlowSummary, b: FlowSummary): number => {
  const aIsRunning = isRunningFlow(a.status);
  const bIsRunning = isRunningFlow(b.status);

  if (aIsRunning && !bIsRunning) return -1;
  if (!aIsRunning && bIsRunning) return 1;

  return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
};

export const compareSteps = (a: Step, b: Step): number => {
  return a.name.localeCompare(b.name);
};

export const upsertFlowList = (
  flows: FlowSummary[],
  flow: FlowSummary
): FlowSummary[] => {
  const existingIndex = flows.findIndex((current) => current.id === flow.id);
  if (existingIndex >= 0) {
    const updatedFlows = [...flows];
    updatedFlows[existingIndex] = flow;
    return updatedFlows;
  }

  return [...flows, flow];
};

export const upsertStepList = (steps: Step[], step: Step): Step[] => {
  const existingIndex = steps.findIndex((current) => current.id === step.id);
  if (existingIndex >= 0) {
    const updatedSteps = [...steps];
    updatedSteps[existingIndex] = step;
    return updatedSteps;
  }

  return [...steps, step].sort(compareSteps);
};

export const updateExistingStepList = (steps: Step[], step: Step): Step[] => {
  const exists = steps.some((current) => current.id === step.id);
  if (!exists) {
    return steps;
  }

  return upsertStepList(steps, step);
};

export const mergeFlowLists = (
  flows: FlowSummary[],
  moreFlows: FlowSummary[]
): FlowSummary[] => {
  return moreFlows.reduce(upsertFlowList, flows).sort(compareFlows);
};

export const mergeResolvedAttributes = (
  current: string[],
  newAttrs?: Record<string, any>
): string[] => {
  if (!newAttrs) return current;

  const outputKeys = Object.keys(newAttrs);
  const hasNewAttrs = outputKeys.some((key) => !current.includes(key));
  if (!hasNewAttrs) return current;

  const resolved = new Set(current);
  outputKeys.forEach((key) => resolved.add(key));
  return Array.from(resolved);
};

export const normalizeFlowAttributes = (
  attrs?: Record<string, any>
): Record<string, AttributeValue> => {
  if (!attrs) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(attrs).map(([name, value]) => {
      if (Array.isArray(value)) {
        return [name, value[value.length - 1]];
      }
      return [name, value];
    })
  );
};

export const toFlowSummary = (item: QueryFlowsItem): FlowSummary => {
  return {
    id: item.id,
    status: item.status,
    timestamp: item.timestamp,
    error: item.error,
  };
};

type FlowSummaryState = {
  id: string;
  status: FlowContext["status"];
  created_at?: string;
  completed_at?: string;
  error?: string;
};

const flowSummaryTimestamp = (state: FlowSummaryState): string => {
  if (isRunningFlow(state.status)) {
    return state.created_at || new Date().toISOString();
  }
  return state.completed_at || state.created_at || new Date().toISOString();
};

export const toFlowSummaryFromState = (
  state: FlowSummaryState
): FlowSummary => {
  return {
    id: state.id,
    status: state.status,
    timestamp: flowSummaryTimestamp(state),
    error: state.error,
  };
};

export const toStepMap = (steps: Step[]): Record<string, Step> => {
  return Object.fromEntries(steps.map((step) => [step.id, step]));
};

export function buildFlowContext(state: FlowStateUpdate): FlowContext {
  let errorState = undefined;
  if (state.error) {
    errorState = {
      message: state.error,
      step_id: "",
      timestamp: new Date().toISOString(),
    };
  }
  let plan = undefined;
  if (state.plan && Object.keys(state.plan.steps || {}).length > 0) {
    plan = state.plan;
  }
  const attrs = normalizeFlowAttributes(state.attributes);
  return {
    id: state.id,
    status: state.status,
    state: attrs,
    error_state: errorState,
    plan,
    started_at: state.created_at || new Date().toISOString(),
    completed_at: state.completed_at,
  };
}

export function buildExecutionList(state: FlowStateUpdate): ExecutionResult[] {
  return Object.entries(state.executions || {}).map(
    ([stepId, exec]: [string, any]) => ({
      step_id: stepId,
      flow_id: state.id,
      status: exec.status || "pending",
      inputs: exec.inputs || {},
      outputs: exec.outputs,
      unsatisfied: exec.unsatisfied,
      error_message: exec.error,
      started_at: exec.started_at || "",
      completed_at: exec.completed_at,
      duration_ms: exec.duration,
      work_items: exec.work_items,
    })
  );
}

export function computeResolvedAttributes(
  stateAttrs: Record<string, any>,
  executions: ExecutionResult[]
): string[] {
  const resolved = new Set<string>(Object.keys(stateAttrs));
  executions.forEach((exec) => {
    if (exec.status === "completed" && exec.outputs) {
      Object.keys(exec.outputs).forEach((attr) => resolved.add(attr));
    }
  });
  return Array.from(resolved);
}
