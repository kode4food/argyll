import { create } from "zustand";
import { devtools } from "zustand/middleware";
import { api, WorkflowContext, ExecutionResult, Step } from "../api";

interface StepHealthInfo {
  health_status: string;
  health_error?: string;
}

const compareWorkflows = (a: WorkflowContext, b: WorkflowContext): number => {
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

interface WorkflowState {
  steps: Step[];
  stepHealth: Record<string, StepHealthInfo>;
  workflows: WorkflowContext[];
  selectedWorkflow: string | null;
  workflowData: WorkflowContext | null;
  executions: ExecutionResult[];
  resolvedAttributes: string[];
  loading: boolean;
  error: string | null;
  workflowNotFound: boolean;
  isWorkflowMode: boolean;
  nextSequence: number;
  loadSteps: () => Promise<void>;
  loadWorkflows: () => Promise<void>;
  addStep: (step: Step) => void;
  removeStep: (stepId: string) => void;
  addWorkflow: (workflow: WorkflowContext) => void;
  removeWorkflow: (workflowId: string) => void;
  selectWorkflow: (workflowId: string | null) => void;
  loadWorkflowData: (workflowId: string) => Promise<void>;
  refreshExecutions: (workflowId: string) => Promise<void>;
  updateWorkflowFromWebSocket: (update: Partial<WorkflowContext>) => void;
  updateWorkflowStatus: (
    workflowId: string,
    status: WorkflowContext["status"],
    completed_at?: string
  ) => void;
  updateStepHealth: (stepId: string, health: string, error?: string) => void;
}

export const useWorkflowStore = create<WorkflowState>()(
  devtools(
    (set, get) => ({
      steps: [],
      stepHealth: {},
      workflows: [],
      selectedWorkflow: null,
      workflowData: null,
      executions: [],
      resolvedAttributes: [],
      loading: false,
      error: null,
      workflowNotFound: false,
      isWorkflowMode: false,
      nextSequence: 0,

      loadSteps: async () => {
        try {
          const engineState = await api.getEngineState();
          const steps = Object.values(engineState.steps || {});
          const healthMap: Record<string, StepHealthInfo> = {};

          Object.entries(engineState.health || {}).forEach(
            ([stepId, health]: [string, any]) => {
              healthMap[stepId] = {
                health_status: health.status || "unknown",
                health_error: health.error,
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

      loadWorkflows: async () => {
        try {
          const workflows = await api.listWorkflows();
          set({
            workflows: (workflows || []).sort(compareWorkflows),
          });
        } catch (error) {
          console.error("Failed to load workflows:", error);
          set({
            error:
              error instanceof Error
                ? error.message
                : "Failed to load workflows",
          });
        }
      },

      selectWorkflow: (workflowId: string | null) => {
        set({
          selectedWorkflow: workflowId,
          workflowNotFound: false,
          error: null,
          workflowData: null,
          executions: [],
          resolvedAttributes: [],
          isWorkflowMode: !!workflowId,
          nextSequence: 0,
        });

        if (workflowId) {
          get().loadWorkflowData(workflowId);
        }
      },

      loadWorkflowData: async (workflowId: string) => {
        set({ loading: true, error: null, workflowNotFound: false });

        try {
          const { workflow, executions } =
            await api.getWorkflowWithEvents(workflowId);

          const resolved = new Set<string>();

          if (workflow?.state) {
            Object.keys(workflow.state).forEach((attr) => {
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
            workflowData: workflow,
            executions: executions || [],
            resolvedAttributes: Array.from(resolved),
            loading: false,
          });
        } catch (error) {
          console.error("Failed to load workflow data:", error);
          set({
            workflowData: null,
            executions: [],
            resolvedAttributes: [],
            workflowNotFound: true,
            loading: false,
          });
        }
      },

      refreshExecutions: async (workflowId: string) => {
        try {
          const executions = await api.getExecutions(workflowId);
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

      addWorkflow: (workflow: WorkflowContext) => {
        const { workflows } = get();
        set({
          workflows: [...workflows, workflow].sort(compareWorkflows),
        });
      },

      removeWorkflow: (workflowId: string) => {
        const { workflows } = get();
        set({ workflows: workflows.filter((w) => w.id !== workflowId) });
      },

      updateWorkflowFromWebSocket: (update: Partial<WorkflowContext>) => {
        const { workflowData, workflows, resolvedAttributes } = get();
        if (workflowData) {
          const updatedWorkflow = { ...workflowData, ...update };

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

          const workflowIndex = workflows.findIndex(
            (w) => w.id === updatedWorkflow.id
          );
          const updatedWorkflows =
            workflowIndex >= 0
              ? workflows.map((w, i) =>
                  i === workflowIndex ? updatedWorkflow : w
                )
              : workflows;

          set({
            workflowData: updatedWorkflow,
            workflows: updatedWorkflows,
            resolvedAttributes: newResolvedAttrs,
          });
        }
      },

      updateWorkflowStatus: (
        workflowId: string,
        status: WorkflowContext["status"],
        completed_at?: string
      ) => {
        const { workflows } = get();
        const workflowIndex = workflows.findIndex((w) => w.id === workflowId);

        if (workflowIndex < 0) {
          return;
        }

        const existingWorkflow = workflows[workflowIndex];

        if (existingWorkflow.status === status) {
          return;
        }

        const updatedWorkflows = workflows.map((w, i) =>
          i === workflowIndex
            ? { ...w, status, ...(completed_at && { completed_at }) }
            : w
        );

        set({
          workflows: updatedWorkflows.sort(compareWorkflows),
        });
      },

      updateStepHealth: (stepId: string, health: string, error?: string) => {
        const { stepHealth } = get();
        set({
          stepHealth: {
            ...stepHealth,
            [stepId]: {
              health_status: health,
              health_error: error,
            },
          },
        });
      },
    }),
    { name: "WorkflowStore" }
  )
);

// State selectors
export const useSteps = () => useWorkflowStore((state) => state.steps);
export const useWorkflows = () => useWorkflowStore((state) => state.workflows);
export const useSelectedWorkflow = () =>
  useWorkflowStore((state) => state.selectedWorkflow);
export const useWorkflowData = () =>
  useWorkflowStore((state) => state.workflowData);
export const useExecutions = () =>
  useWorkflowStore((state) => state.executions);
export const useResolvedAttributes = () =>
  useWorkflowStore((state) => state.resolvedAttributes);
export const useWorkflowLoading = () =>
  useWorkflowStore((state) => state.loading);
export const useWorkflowError = () => useWorkflowStore((state) => state.error);
export const useIsWorkflowMode = () =>
  useWorkflowStore((state) => state.isWorkflowMode);

// Action selectors
type ActionKeys =
  | "loadSteps"
  | "loadWorkflows"
  | "addStep"
  | "removeStep"
  | "addWorkflow"
  | "removeWorkflow"
  | "selectWorkflow"
  | "refreshExecutions"
  | "updateWorkflowFromWebSocket"
  | "updateWorkflowStatus"
  | "updateStepHealth";

const createActionHook =
  <K extends ActionKeys>(key: K) =>
  () =>
    useWorkflowStore((state) => state[key]);

export const useLoadSteps = createActionHook("loadSteps");
export const useLoadWorkflows = createActionHook("loadWorkflows");
export const useAddStep = createActionHook("addStep");
export const useRemoveStep = createActionHook("removeStep");
export const useAddWorkflow = createActionHook("addWorkflow");
export const useRemoveWorkflow = createActionHook("removeWorkflow");
export const useSelectWorkflow = createActionHook("selectWorkflow");
export const useRefreshExecutions = createActionHook("refreshExecutions");
export const useUpdateWorkflowFromWebSocket = createActionHook(
  "updateWorkflowFromWebSocket"
);
export const useUpdateWorkflowStatus = createActionHook("updateWorkflowStatus");
export const useUpdateStepHealth = createActionHook("updateStepHealth");
