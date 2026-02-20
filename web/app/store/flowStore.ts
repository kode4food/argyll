import { create } from "zustand";
import { devtools } from "zustand/middleware";
import {
  api,
  FlowContext,
  ExecutionPlan,
  ExecutionResult,
  Step,
  WorkState,
} from "../api";
import { ConnectionStatus } from "../types/websocket";

interface StepHealthInfo {
  status: string;
  error?: string;
}

declare global {
  interface Window {
    flowStore?: typeof useFlowStore;
  }
}

const FlowListPageSize = 1000;

const compareFlows = (a: FlowContext, b: FlowContext): number => {
  const aIsActive = a.status === "active";
  const bIsActive = b.status === "active";

  if (aIsActive && !bIsActive) return -1;
  if (!aIsActive && bIsActive) return 1;

  if (aIsActive && bIsActive) {
    return new Date(b.started_at).getTime() - new Date(a.started_at).getTime();
  } else {
    const aTime = a.completed_at || a.started_at;
    const bTime = b.completed_at || b.started_at;
    return new Date(bTime).getTime() - new Date(aTime).getTime();
  }
};

const compareSteps = (a: Step, b: Step): number => {
  return a.name.localeCompare(b.name);
};

const upsertStepList = (steps: Step[], step: Step): Step[] => {
  const existingIndex = steps.findIndex((current) => current.id === step.id);
  if (existingIndex >= 0) {
    const updatedSteps = [...steps];
    updatedSteps[existingIndex] = step;
    return updatedSteps;
  }

  return [...steps, step].sort(compareSteps);
};

const updateExistingStepList = (steps: Step[], step: Step): Step[] => {
  const exists = steps.some((current) => current.id === step.id);
  if (!exists) {
    return steps;
  }

  return upsertStepList(steps, step);
};

const mergeResolvedAttributes = (
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

interface FlowState {
  steps: Step[];
  stepHealth: Record<string, StepHealthInfo>;
  flows: FlowContext[];
  flowsCursor: string | null;
  flowsHasMore: boolean;
  flowsLoading: boolean;
  selectedFlow: string | null;
  flowData: FlowContext | null;
  executions: ExecutionResult[];
  resolvedAttributes: string[];
  loading: boolean;
  error: string | null;
  flowNotFound: boolean;
  engineConnectionStatus: ConnectionStatus;
  engineReconnectAttempt: number;
  engineReconnectRequest: number;
  loadSteps: () => Promise<void>;
  loadFlows: () => Promise<void>;
  loadMoreFlows: () => Promise<void>;
  addStep: (step: Step) => void;
  upsertStep: (step: Step) => void;
  updateStep: (step: Step) => void;
  removeStep: (stepId: string) => void;
  addFlow: (flow: FlowContext) => void;
  removeFlow: (flowId: string) => void;
  selectFlow: (flowId: string | null) => void;
  updateFlowFromWebSocket: (update: Partial<FlowContext>) => void;
  updateFlowStatus: (
    flowId: string,
    status: FlowContext["status"],
    completed_at?: string
  ) => void;
  updateStepHealth: (stepId: string, health: string, error?: string) => void;
  initializeExecutions: (flowId: string, plan: ExecutionPlan) => void;
  updateExecution: (stepId: string, updates: Partial<ExecutionResult>) => void;
  updateWorkItem: (
    stepId: string,
    token: string,
    updates: Partial<WorkState>,
    retryToken?: string
  ) => void;
  setEngineSocketStatus: (
    status: ConnectionStatus,
    reconnectAttempt: number
  ) => void;
  requestEngineReconnect: () => void;
  setCatalogState: (steps: Record<string, Step>) => void;
  setPartitionState: (health: Record<string, StepHealthInfo>) => void;
  setFlowState: (state: {
    id: string;
    status: FlowContext["status"];
    attributes?: Record<string, any>;
    plan?: ExecutionPlan;
    executions?: Record<string, any>;
    created_at?: string;
    completed_at?: string;
    error?: string;
  }) => void;
}

export const useFlowStore = create<FlowState>()(
  devtools(
    (set, get) => ({
      steps: [],
      stepHealth: {},
      flows: [],
      flowsCursor: null,
      flowsHasMore: false,
      flowsLoading: false,
      selectedFlow: null,
      flowData: null,
      executions: [],
      resolvedAttributes: [],
      loading: false,
      error: null,
      flowNotFound: false,
      engineConnectionStatus: "connecting",
      engineReconnectAttempt: 0,
      engineReconnectRequest: 0,

      loadSteps: async () => {
        try {
          const engineData = await api.getEngine();
          get().setCatalogState(engineData.steps || {});
          get().setPartitionState(engineData.health || {});
        } catch (error) {
          console.error("Failed to load steps:", error);
          set({
            error:
              error instanceof Error ? error.message : "Failed to load steps",
          });
        }
      },

      loadFlows: async () => {
        if (get().flowsLoading) {
          return;
        }
        try {
          set({ flowsLoading: true });
          const resp = await api.listFlowsPage({ limit: FlowListPageSize });
          const flows = (resp.flows || []).map((item) => ({
            id: item.id,
            status: item.digest?.status || "active",
            state: {},
            error_state: item.digest?.error
              ? {
                  message: item.digest.error,
                  step_id: "",
                  timestamp: new Date().toISOString(),
                }
              : undefined,
            plan: undefined,
            started_at: item.digest?.created_at || new Date().toISOString(),
            completed_at: item.digest?.completed_at,
          }));
          set({
            flows: flows.sort(compareFlows),
            flowsCursor: resp.next_cursor ?? null,
            flowsHasMore: resp.has_more ?? false,
          });
        } catch (error) {
          console.error("Failed to load flows:", error);
          set({
            error:
              error instanceof Error ? error.message : "Failed to load flows",
          });
        } finally {
          set({ flowsLoading: false });
        }
      },

      loadMoreFlows: async () => {
        const { flowsLoading, flowsHasMore, flowsCursor, flows } = get();
        if (flowsLoading || !flowsHasMore) {
          return;
        }
        try {
          set({ flowsLoading: true });
          const resp = await api.listFlowsPage({
            limit: FlowListPageSize,
            cursor: flowsCursor ?? undefined,
          });
          const moreFlows = (resp.flows || []).map((item) => ({
            id: item.id,
            status: item.digest?.status || "active",
            state: {},
            error_state: item.digest?.error
              ? {
                  message: item.digest.error,
                  step_id: "",
                  timestamp: new Date().toISOString(),
                }
              : undefined,
            plan: undefined,
            started_at: item.digest?.created_at || new Date().toISOString(),
            completed_at: item.digest?.completed_at,
          }));
          const merged = [...flows, ...moreFlows];
          set({
            flows: merged.sort(compareFlows),
            flowsCursor: resp.next_cursor ?? flowsCursor,
            flowsHasMore: resp.has_more ?? false,
          });
        } catch (error) {
          console.error("Failed to load more flows:", error);
          set({
            error:
              error instanceof Error ? error.message : "Failed to load flows",
          });
        } finally {
          set({ flowsLoading: false });
        }
      },

      selectFlow: (flowId: string | null) => {
        const { selectedFlow: currentSelected, flowData: currentFlowData } =
          get();

        if (
          flowId &&
          flowId === currentSelected &&
          currentFlowData?.id === flowId
        ) {
          return;
        }

        set({
          selectedFlow: flowId,
          flowNotFound: false,
          error: null,
          flowData: null,
          executions: [],
          resolvedAttributes: [],
          loading: !!flowId,
        });
      },

      addStep: (step: Step) => {
        set((state) => ({
          steps: upsertStepList(state.steps, step),
        }));
      },

      upsertStep: (step: Step) => {
        set((state) => ({
          steps: upsertStepList(state.steps, step),
        }));
      },

      updateStep: (step: Step) => {
        set((state) => {
          const updatedSteps = updateExistingStepList(state.steps, step);
          if (updatedSteps === state.steps) {
            return state;
          }
          return { steps: updatedSteps };
        });
      },

      removeStep: (stepId: string) => {
        const { steps, stepHealth } = get();
        const { [stepId]: removed, ...newHealthRecord } = stepHealth;
        set({
          steps: steps.filter((s) => s.id !== stepId),
          stepHealth: newHealthRecord,
        });
      },

      addFlow: (flow: FlowContext) => {
        const { flows } = get();
        set({
          flows: [...flows, flow].sort(compareFlows),
        });
      },

      removeFlow: (flowId: string) => {
        const { flows } = get();
        set({ flows: flows.filter((w) => w.id !== flowId) });
      },

      updateFlowFromWebSocket: (update: Partial<FlowContext>) => {
        const { flowData, flows, resolvedAttributes } = get();
        if (flowData) {
          const updatedFlow = { ...flowData, ...update };
          const newResolvedAttrs = mergeResolvedAttributes(
            resolvedAttributes,
            update.state
          );

          const flowIndex = flows.findIndex((w) => w.id === updatedFlow.id);
          const updatedFlows =
            flowIndex >= 0
              ? flows.map((w, i) => (i === flowIndex ? updatedFlow : w))
              : flows;

          set({
            flowData: updatedFlow,
            flows: updatedFlows,
            resolvedAttributes: newResolvedAttrs,
          });
        }
      },

      updateFlowStatus: (
        flowId: string,
        status: FlowContext["status"],
        completed_at?: string
      ) => {
        const { flows } = get();
        const flowIndex = flows.findIndex((w) => w.id === flowId);

        if (flowIndex < 0) {
          return;
        }

        const existingFlow = flows[flowIndex];

        if (existingFlow.status === status) {
          return;
        }

        const updatedFlows = flows.map((w, i) =>
          i === flowIndex
            ? { ...w, status, ...(completed_at && { completed_at }) }
            : w
        );

        set({
          flows: updatedFlows.sort(compareFlows),
        });
      },

      updateStepHealth: (stepId: string, health: string, error?: string) => {
        const { stepHealth } = get();
        set({
          stepHealth: {
            ...stepHealth,
            [stepId]: {
              status: health,
              error: error,
            },
          },
        });
      },

      initializeExecutions: (flowId: string, plan: ExecutionPlan) => {
        if (!plan?.steps) {
          set({ executions: [] });
          return;
        }

        const executions: ExecutionResult[] = Object.keys(plan.steps).map(
          (stepId) => ({
            step_id: stepId,
            flow_id: flowId,
            status: "pending",
            inputs: {},
            started_at: "",
          })
        );

        set({ executions });
      },

      updateExecution: (stepId: string, updates: Partial<ExecutionResult>) => {
        const { executions, resolvedAttributes } = get();
        const index = executions.findIndex((e) => e.step_id === stepId);
        if (index >= 0) {
          const updated = [...executions];
          updated[index] = { ...updated[index], ...updates };

          const newResolvedAttrs = mergeResolvedAttributes(
            resolvedAttributes,
            updates.outputs
          );

          set({ executions: updated, resolvedAttributes: newResolvedAttrs });
        }
      },

      updateWorkItem: (
        stepId: string,
        token: string,
        updates: Partial<WorkState>,
        retryToken?: string
      ) => {
        const { executions } = get();
        const index = executions.findIndex((e) => e.step_id === stepId);
        if (index < 0) return;

        const execution = executions[index];
        const workItems = execution.work_items || {};

        // If retry_token exists and differs from token, this is a retry
        // Remove the old token and update under the retry_token
        if (retryToken && retryToken !== token) {
          const existingItem = workItems[token] || {
            token: retryToken,
            status: "pending",
            inputs: {},
            retry_count: 0,
          };

          const { [token]: removed, ...remainingItems } = workItems;

          const updated = [...executions];
          updated[index] = {
            ...execution,
            work_items: {
              ...remainingItems,
              [retryToken]: { ...existingItem, ...updates, token: retryToken },
            },
          };

          set({ executions: updated });
        } else {
          // Normal case: just update the work item with the current token
          const existingItem = workItems[token] || {
            token,
            status: "pending",
            inputs: {},
            retry_count: 0,
          };

          const updated = [...executions];
          updated[index] = {
            ...execution,
            work_items: {
              ...workItems,
              [token]: { ...existingItem, ...updates },
            },
          };

          set({ executions: updated });
        }
      },

      setEngineSocketStatus: (
        status: ConnectionStatus,
        reconnectAttempt: number
      ) => {
        set({
          engineConnectionStatus: status,
          engineReconnectAttempt: reconnectAttempt,
        });
      },

      requestEngineReconnect: () => {
        set((state) => ({
          engineReconnectRequest: state.engineReconnectRequest + 1,
        }));
      },

      setCatalogState: (steps) => {
        set({ steps: Object.values(steps).sort(compareSteps) });
      },

      setPartitionState: (health) => {
        const healthMap: Record<string, StepHealthInfo> = {};
        Object.entries(health).forEach(([stepId, h]: [string, any]) => {
          healthMap[stepId] = {
            status: h.status || "unknown",
            error: h.error,
          };
        });
        set({ stepHealth: healthMap });
      },

      setFlowState: (state) => {
        const { selectedFlow } = get();
        if (!selectedFlow || state.id !== selectedFlow) {
          return;
        }

        let errorState = undefined;
        if (state.error) {
          errorState = {
            message: state.error,
            step_id: "",
            timestamp: new Date().toISOString(),
          };
        }

        let executionPlan = undefined;
        if (state.plan && Object.keys(state.plan.steps || {}).length > 0) {
          executionPlan = state.plan;
        }

        const flowData: FlowContext = {
          id: state.id,
          status: state.status,
          state: state.attributes || {},
          error_state: errorState,
          plan: executionPlan,
          started_at: state.created_at || new Date().toISOString(),
          completed_at: state.completed_at,
        };

        const executions: ExecutionResult[] = Object.entries(
          state.executions || {}
        ).map(([stepId, exec]: [string, any]) => ({
          step_id: stepId,
          flow_id: state.id,
          status: exec.status || "pending",
          inputs: exec.inputs || {},
          outputs: exec.outputs,
          error_message: exec.error,
          started_at: exec.started_at || "",
          completed_at: exec.completed_at,
          duration_ms: exec.duration,
          work_items: exec.work_items,
        }));

        const resolved = new Set<string>();
        if (state.attributes) {
          Object.keys(state.attributes).forEach((attr) => resolved.add(attr));
        }
        executions.forEach((exec) => {
          if (exec.status === "completed" && exec.outputs) {
            Object.keys(exec.outputs).forEach((attr) => resolved.add(attr));
          }
        });

        set({
          flowData,
          executions,
          resolvedAttributes: Array.from(resolved),
          loading: false,
          flowNotFound: false,
        });
      },
    }),
    { name: "FlowStore" }
  )
);

const isDevHost =
  typeof window !== "undefined" &&
  (window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1");

if (isDevHost) {
  window.flowStore = useFlowStore;
}

// State selectors
export const useSteps = () => useFlowStore((state) => state.steps);
export const useFlows = () => useFlowStore((state) => state.flows);
export const useFlowsHasMore = () =>
  useFlowStore((state) => state.flowsHasMore);
export const useFlowsLoading = () =>
  useFlowStore((state) => state.flowsLoading);
export const useSelectedFlow = () =>
  useFlowStore((state) => state.selectedFlow);
export const useFlowData = () => useFlowStore((state) => state.flowData);
export const useExecutions = () => useFlowStore((state) => state.executions);
export const useResolvedAttributes = () =>
  useFlowStore((state) => state.resolvedAttributes);
export const useFlowLoading = () => useFlowStore((state) => state.loading);
export const useFlowError = () => useFlowStore((state) => state.error);
export const useFlowNotFound = () =>
  useFlowStore((state) => state.flowNotFound);
export const useEngineConnectionStatus = () =>
  useFlowStore((state) => state.engineConnectionStatus);
export const useEngineReconnectAttempt = () =>
  useFlowStore((state) => state.engineReconnectAttempt);
export const useRequestEngineReconnect = () =>
  useFlowStore((state) => state.requestEngineReconnect);

// Action selectors
type ActionKeys =
  | "loadSteps"
  | "loadFlows"
  | "loadMoreFlows"
  | "addFlow"
  | "removeFlow"
  | "selectFlow"
  | "updateFlowStatus";

const createActionHook =
  <K extends ActionKeys>(key: K) =>
  () =>
    useFlowStore((state) => state[key]);

export const useLoadSteps = createActionHook("loadSteps");
export const useLoadFlows = createActionHook("loadFlows");
export const useLoadMoreFlows = createActionHook("loadMoreFlows");
export const useAddFlow = createActionHook("addFlow");
export const useRemoveFlow = createActionHook("removeFlow");
export const useSelectFlow = createActionHook("selectFlow");
export const useUpdateFlowStatus = createActionHook("updateFlowStatus");
