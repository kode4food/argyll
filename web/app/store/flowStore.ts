import { create } from "zustand";
import { devtools } from "zustand/middleware";
import {
  api,
  ExecutionPlan,
  ExecutionResult,
  FlowContext,
  FlowSummary,
  Step,
  WorkState,
} from "../api";
import { ConnectionStatus } from "../types/websocket";
import {
  buildExecutionList,
  buildFlowContext,
  compareFlows,
  compareSteps,
  computeResolvedAttributes,
  FlowStateUpdate,
  mergeResolvedAttributes,
  toFlowSummaryFromState,
  toStepMap,
  updateExistingStepList,
  upsertFlowList,
  upsertStepList,
} from "./flowStoreHelpers";
import { loadFlowsImpl, loadMoreFlowsImpl } from "./flowStoreLoaders";
import {
  NodeHealth,
  sortNodeIds,
  StepHealthInfo,
  toStepHealthMap,
} from "./flowStoreHealthHelpers";

export type { NodeHealth, StepHealthInfo };

export interface StepRef {
  nodeId: string;
  stepId: string;
}

declare global {
  interface Window {
    flowStore?: typeof useFlowStore;
  }
}

interface FlowState {
  steps: Step[];
  healthByNode: Record<string, Record<string, StepHealthInfo>>;
  stepHealth: Record<string, StepHealthInfo>;
  healthNodeIds: string[];
  flows: FlowSummary[];
  visibleFlowIDs: string[];
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
  addFlow: (flow: FlowSummary) => void;
  removeFlow: (flowId: string) => void;
  selectFlow: (flowId: string | null) => void;
  setFlowNotFound: (flowId: string) => void;
  updateFlowData: (update: Partial<FlowContext>) => void;
  updateStepHealth: (ref: StepRef, health: string, error?: string) => void;
  initializeExecutions: (flowId: string, plan: ExecutionPlan) => void;
  updateExecution: (stepId: string, updates: Partial<ExecutionResult>) => void;
  updateWorkItem: (
    stepId: string,
    token: string,
    updates: Partial<WorkState>
  ) => void;
  setEngineSocketStatus: (
    status: ConnectionStatus,
    reconnectAttempt: number
  ) => void;
  requestEngineReconnect: () => void;
  setVisibleFlowIDs: (flowIDs: string[]) => void;
  setCatalogState: (steps: Record<string, Step>) => void;
  setHealthState: (
    healthByNode: Record<string, Record<string, StepHealthInfo>>
  ) => void;
  setFlowState: (state: FlowStateUpdate) => void;
}

export const useFlowStore = create<FlowState>()(
  devtools(
    (set, get) => ({
      steps: [],
      healthByNode: {},
      stepHealth: {},
      healthNodeIds: [],
      flows: [],
      visibleFlowIDs: [],
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
          const healthByNode: Record<string, Record<string, any>> = {};
          for (const [nodeId, node] of Object.entries(
            engineData.health || {}
          )) {
            healthByNode[nodeId] = node.health ?? {};
          }
          get().setHealthState(healthByNode);
        } catch (error) {
          console.error("Failed to load steps:", error);
          set({
            error:
              error instanceof Error ? error.message : "Failed to load steps",
          });
        }
      },

      loadFlows: async () => {
        if (get().flowsLoading) return;
        await loadFlowsImpl((update) => set(update));
      },

      loadMoreFlows: async () => {
        const { flowsLoading, flowsHasMore, flowsCursor, flows } = get();
        if (flowsLoading || !flowsHasMore) return;
        await loadMoreFlowsImpl((update) => set(update), flowsCursor, flows);
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

      setFlowNotFound: (flowId: string) => {
        if (get().selectedFlow !== flowId) {
          return;
        }

        set({
          flowData: null,
          executions: [],
          resolvedAttributes: [],
          loading: false,
          flowNotFound: true,
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

      addFlow: (flow: FlowSummary) => {
        set({
          flows: upsertFlowList(get().flows, flow).sort(compareFlows),
        });
      },

      removeFlow: (flowId: string) => {
        const { flows } = get();
        set({ flows: flows.filter((w) => w.id !== flowId) });
      },

      updateFlowData: (update: Partial<FlowContext>) => {
        const { flowData, resolvedAttributes } = get();
        if (!flowData) {
          return;
        }

        const updatedFlow = { ...flowData, ...update };
        const newResolvedAttrs = mergeResolvedAttributes(
          resolvedAttributes,
          update.state
        );

        set({
          flowData: updatedFlow,
          resolvedAttributes: newResolvedAttrs,
        });
      },

      updateStepHealth: (ref: StepRef, health: string, error?: string) => {
        const { nodeId, stepId } = ref;
        const { healthByNode, healthNodeIds, steps } = get();
        const stepsById = toStepMap(steps);
        const nextNodeIds = healthNodeIds.includes(nodeId)
          ? healthNodeIds
          : sortNodeIds([...healthNodeIds, nodeId]);
        const nextHealthByNode = {
          ...healthByNode,
          [nodeId]: {
            ...(healthByNode[nodeId] || {}),
            [stepId]: {
              status: health,
              ...(error && { error }),
            },
          },
        };
        set({
          healthByNode: nextHealthByNode,
          healthNodeIds: nextNodeIds,
          stepHealth: toStepHealthMap(nextHealthByNode, stepsById),
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
        updates: Partial<WorkState>
      ) => {
        const { executions } = get();
        const index = executions.findIndex((e) => e.step_id === stepId);
        if (index < 0) return;

        const execution = executions[index];
        const workItems = execution.work_items || {};
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

      setVisibleFlowIDs: (flowIDs: string[]) => {
        set({ visibleFlowIDs: flowIDs });
      },

      setCatalogState: (steps) => {
        const nextSteps = Object.values(steps).sort(compareSteps);
        set({
          steps: nextSteps,
          stepHealth: toStepHealthMap(get().healthByNode, toStepMap(nextSteps)),
        });
      },

      setHealthState: (healthByNode) => {
        const stepsById = toStepMap(get().steps);
        set({
          healthByNode,
          healthNodeIds: sortNodeIds(Object.keys(healthByNode)),
          stepHealth: toStepHealthMap(healthByNode, stepsById),
        });
      },

      setFlowState: (state) => {
        const { selectedFlow, flows } = get();
        if (!selectedFlow || state.id !== selectedFlow) {
          return;
        }
        const flowData = buildFlowContext(state);
        const executions = buildExecutionList(state);
        set({
          flowData,
          flows: upsertFlowList(flows, toFlowSummaryFromState(state)).sort(
            compareFlows
          ),
          executions,
          resolvedAttributes: computeResolvedAttributes(
            flowData.state ?? {},
            executions
          ),
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
  | "setVisibleFlowIDs";

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
export const useSetVisibleFlowIDs = createActionHook("setVisibleFlowIDs");
