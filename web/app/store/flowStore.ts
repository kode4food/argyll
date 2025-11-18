import { create } from "zustand";
import { devtools } from "zustand/middleware";
import { api, FlowContext, ExecutionResult, Step } from "../api";

interface StepHealthInfo {
  status: string;
  error?: string;
}

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

interface FlowState {
  steps: Step[];
  stepHealth: Record<string, StepHealthInfo>;
  flows: FlowContext[];
  selectedFlow: string | null;
  flowData: FlowContext | null;
  executions: ExecutionResult[];
  resolvedAttributes: string[];
  loading: boolean;
  error: string | null;
  flowNotFound: boolean;
  isFlowMode: boolean;
  nextSequence: number;
  loadSteps: () => Promise<void>;
  loadFlows: () => Promise<void>;
  addStep: (step: Step) => void;
  removeStep: (stepId: string) => void;
  addFlow: (flow: FlowContext) => void;
  removeFlow: (flowId: string) => void;
  selectFlow: (flowId: string | null) => void;
  loadFlowData: (flowId: string) => Promise<void>;
  refreshExecutions: (flowId: string) => Promise<void>;
  updateFlowFromWebSocket: (update: Partial<FlowContext>) => void;
  updateFlowStatus: (
    flowId: string,
    status: FlowContext["status"],
    completed_at?: string
  ) => void;
  updateStepHealth: (stepId: string, health: string, error?: string) => void;
}

export const useFlowStore = create<FlowState>()(
  devtools(
    (set, get) => ({
      steps: [],
      stepHealth: {},
      flows: [],
      selectedFlow: null,
      flowData: null,
      executions: [],
      resolvedAttributes: [],
      loading: false,
      error: null,
      flowNotFound: false,
      isFlowMode: false,
      nextSequence: 0,

      loadSteps: async () => {
        try {
          const engineState = await api.getEngineState();
          const steps = Object.values(engineState.steps || {});
          const healthMap: Record<string, StepHealthInfo> = {};

          Object.entries(engineState.health || {}).forEach(
            ([stepId, health]: [string, any]) => {
              healthMap[stepId] = {
                status: health.status || "unknown",
                error: health.error,
              };
            }
          );

          set({
            steps: (steps || []).sort((a, b) => a.name.localeCompare(b.name)),
            stepHealth: healthMap,
          });
        } catch (error) {
          console.error("Failed to load steps:", error);
          set({
            error:
              error instanceof Error ? error.message : "Failed to load steps",
          });
        }
      },

      loadFlows: async () => {
        try {
          const flows = await api.listFlows();
          set({
            flows: (flows || []).sort(compareFlows),
          });
        } catch (error) {
          console.error("Failed to load flows:", error);
          set({
            error:
              error instanceof Error
                ? error.message
                : "Failed to load flows",
          });
        }
      },

      selectFlow: (flowId: string | null) => {
        set({
          selectedFlow: flowId,
          flowNotFound: false,
          error: null,
          flowData: null,
          executions: [],
          resolvedAttributes: [],
          isFlowMode: !!flowId,
          nextSequence: 0,
        });

        if (flowId) {
          get().loadFlowData(flowId);
        }
      },

      loadFlowData: async (flowId: string) => {
        set({ loading: true, error: null, flowNotFound: false });

        try {
          const { flow, executions } =
            await api.getFlowWithEvents(flowId);

          const resolved = new Set<string>();

          if (flow?.state) {
            Object.keys(flow.state).forEach((attr) => {
              resolved.add(attr);
            });
          }

          executions.forEach((exec) => {
            if (exec.status === "completed" && exec.outputs) {
              Object.keys(exec.outputs).forEach((attr) => {
                resolved.add(attr);
              });
            }
          });

          set({
            flowData: flow,
            executions: executions || [],
            resolvedAttributes: Array.from(resolved),
            loading: false,
          });
        } catch (error) {
          console.error("Failed to load flow data:", error);
          set({
            flowData: null,
            executions: [],
            resolvedAttributes: [],
            flowNotFound: true,
            loading: false,
          });
        }
      },

      refreshExecutions: async (flowId: string) => {
        try {
          const executions = await api.getExecutions(flowId);
          set({ executions: executions || [] });
        } catch (error) {
          console.error("Failed to refresh executions:", error);
        }
      },

      addStep: (step: Step) => {
        const { steps } = get();
        const existingIndex = steps.findIndex((s) => s.id === step.id);

        if (existingIndex >= 0) {
          const updatedSteps = [...steps];
          updatedSteps[existingIndex] = step;
          set({ steps: updatedSteps });
        } else {
          const newSteps = [...steps, step];
          newSteps.sort((a, b) => a.name.localeCompare(b.name));
          set({ steps: newSteps });
        }
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

          let newResolvedAttrs = resolvedAttributes;
          if (update.state) {
            const stateKeys = Object.keys(update.state);
            const hasNewAttrs = stateKeys.some(
              (key) => !resolvedAttributes.includes(key)
            );
            if (hasNewAttrs) {
              const resolved = new Set(resolvedAttributes);
              stateKeys.forEach((key) => resolved.add(key));
              newResolvedAttrs = Array.from(resolved);
            }
          }

          const flowIndex = flows.findIndex(
            (w) => w.id === updatedFlow.id
          );
          const updatedFlows =
            flowIndex >= 0
              ? flows.map((w, i) =>
                  i === flowIndex ? updatedFlow : w
                )
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
    }),
    { name: "FlowStore" }
  )
);

// State selectors
export const useSteps = () => useFlowStore((state) => state.steps);
export const useFlows = () => useFlowStore((state) => state.flows);
export const useSelectedFlow = () =>
  useFlowStore((state) => state.selectedFlow);
export const useFlowData = () =>
  useFlowStore((state) => state.flowData);
export const useExecutions = () =>
  useFlowStore((state) => state.executions);
export const useResolvedAttributes = () =>
  useFlowStore((state) => state.resolvedAttributes);
export const useFlowLoading = () =>
  useFlowStore((state) => state.loading);
export const useFlowError = () => useFlowStore((state) => state.error);
export const useIsFlowMode = () =>
  useFlowStore((state) => state.isFlowMode);

// Action selectors
type ActionKeys =
  | "loadSteps"
  | "loadFlows"
  | "addStep"
  | "removeStep"
  | "addFlow"
  | "removeFlow"
  | "selectFlow"
  | "refreshExecutions"
  | "updateFlowFromWebSocket"
  | "updateFlowStatus"
  | "updateStepHealth";

const createActionHook =
  <K extends ActionKeys>(key: K) =>
  () =>
    useFlowStore((state) => state[key]);

export const useLoadSteps = createActionHook("loadSteps");
export const useLoadFlows = createActionHook("loadFlows");
export const useAddStep = createActionHook("addStep");
export const useRemoveStep = createActionHook("removeStep");
export const useAddFlow = createActionHook("addFlow");
export const useRemoveFlow = createActionHook("removeFlow");
export const useSelectFlow = createActionHook("selectFlow");
export const useRefreshExecutions = createActionHook("refreshExecutions");
export const useUpdateFlowFromWebSocket = createActionHook(
  "updateFlowFromWebSocket"
);
export const useUpdateFlowStatus = createActionHook("updateFlowStatus");
export const useUpdateStepHealth = createActionHook("updateStepHealth");
