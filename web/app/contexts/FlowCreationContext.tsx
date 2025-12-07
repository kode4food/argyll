import React, {
  createContext,
  useContext,
  useState,
  useRef,
  useCallback,
  useEffect,
} from "react";
import { useRouter } from "next/navigation";
import { Step } from "../api";
import {
  useSteps,
  useLoadFlows,
  useAddFlow,
  useRemoveFlow,
} from "../store/flowStore";
import { useUI } from "../contexts/UIContext";
import { useThrottledValue } from "../hooks/useThrottledValue";
import { api } from "../api";
import {
  parseState,
  filterDefaultValues,
  addRequiredDefaults,
} from "@/utils/stateUtils";
import { generateFlowId, generatePadded } from "@/utils/flowUtils";
import { sortStepsByType } from "@/utils/stepUtils";
import toast from "react-hot-toast";

export interface FlowCreationContextValue {
  newID: string;
  setNewID: (id: string) => void;
  setIDManuallyEdited: (edited: boolean) => void;
  handleStepChange: (stepIds: string[]) => void;
  initialState: string;
  setInitialState: (state: string) => void;
  creating: boolean;
  handleCreateFlow: () => void;
  steps: Step[];
  generateID: () => string;
  sortSteps: (steps: Step[]) => Step[];
}

const FlowCreationContext = createContext<FlowCreationContextValue | null>(
  null
);

export const FlowCreationProvider = ({
  value,
  children,
}: {
  value: FlowCreationContextValue;
  children: React.ReactNode;
}) => (
  <FlowCreationContext.Provider value={value}>
    {children}
  </FlowCreationContext.Provider>
);

export const useFlowCreation = (): FlowCreationContextValue => {
  const ctx = useContext(FlowCreationContext);
  if (!ctx) {
    throw new Error("useFlowCreation must be used within FlowCreationProvider");
  }
  return ctx;
};

export const FlowCreationStateProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const router = useRouter();
  const steps = useSteps();
  const loadFlows = useLoadFlows();
  const addFlow = useAddFlow();
  const removeFlow = useRemoveFlow();
  const {
    previewPlan,
    updatePreviewPlan,
    clearPreviewPlan,
    setSelectedStep,
    goalStepIds,
    setGoalStepIds,
    showCreateForm,
    setShowCreateForm,
  } = useUI();

  const [newID, setNewID] = useState("");
  const [initialState, setInitialState] = useState("{}");
  const [creating, setCreating] = useState(false);
  const [idManuallyEdited, setIDManuallyEdited] = useState(false);
  const initializedGoalsRef = useRef(false);
  const prevShowCreateFormRef = useRef(showCreateForm);

  const resetForm = useCallback(() => {
    setNewID("");
    setGoalStepIds([]);
    setSelectedStep(null);
    setInitialState("{}");
    setIDManuallyEdited(false);
    clearPreviewPlan();
    initializedGoalsRef.current = false;
  }, [clearPreviewPlan, setGoalStepIds, setSelectedStep]);

  const handleStepChange = useCallback(
    async (stepIds: string[]) => {
      const currentState = parseState(initialState);
      const nonDefaultState = filterDefaultValues(currentState, steps);

      if (stepIds.length === 0) {
        setInitialState(JSON.stringify(nonDefaultState, null, 2));
        clearPreviewPlan();
        setGoalStepIds([]);
        return;
      }

      try {
        const executionPlan = await api.getExecutionPlan(
          stepIds,
          nonDefaultState
        );

        const stateWithDefaults = addRequiredDefaults(
          nonDefaultState,
          executionPlan
        );

        setInitialState(JSON.stringify(stateWithDefaults, null, 2));

        if (!idManuallyEdited) {
          const lastGoalId = stepIds[stepIds.length - 1];
          const goalStep = steps.find((s) => s.id === lastGoalId);
          const goalName = goalStep?.name || lastGoalId;
          const kebabName = goalName
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-+|-+$/g, "");
          setNewID(`${kebabName}-${generatePadded()}`);
        }

        if (stepIds.length > 1) {
          const lastGoal = stepIds[stepIds.length - 1];
          const previousGoals = stepIds.slice(0, -1);

          try {
            const lastGoalPlan = await api.getExecutionPlan([lastGoal], {});
            const lastGoalStepIds = new Set(
              Object.keys(lastGoalPlan.steps || {})
            );

            const remainingGoals = previousGoals.filter(
              (id) => !lastGoalStepIds.has(id)
            );

            const finalGoals = [...remainingGoals, lastGoal];

            if (finalGoals.length !== stepIds.length) {
              setGoalStepIds(finalGoals);
              await updatePreviewPlan(finalGoals, stateWithDefaults);
              return;
            }
          } catch {}
        }

        setGoalStepIds(stepIds);
        await updatePreviewPlan(stepIds, stateWithDefaults);
      } catch (error) {
        clearPreviewPlan();
        setGoalStepIds(stepIds);
      }
    },
    [
      initialState,
      idManuallyEdited,
      steps,
      setGoalStepIds,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  const throttledInitialState = useThrottledValue(initialState, 500);

  useEffect(() => {
    if (!showCreateForm || goalStepIds.length === 0) {
      return;
    }

    const currentState = parseState(throttledInitialState);
    const nonDefaultState = filterDefaultValues(currentState, steps);

    if (Object.keys(currentState).length >= 0) {
      updatePreviewPlan(goalStepIds, nonDefaultState).catch(() => {});
    }
  }, [
    throttledInitialState,
    showCreateForm,
    goalStepIds,
    steps,
    updatePreviewPlan,
  ]);

  useEffect(() => {
    if (!showCreateForm && prevShowCreateFormRef.current) {
      resetForm();
    }
    prevShowCreateFormRef.current = showCreateForm;
  }, [showCreateForm, resetForm]);

  useEffect(() => {
    if (!showCreateForm) {
      return;
    }

    router.prefetch("/flow/placeholder");

    if (goalStepIds.length === 0) {
      initializedGoalsRef.current = false;
      return;
    }

    if (!initializedGoalsRef.current) {
      initializedGoalsRef.current = true;
      handleStepChange(goalStepIds);
    }
  }, [showCreateForm, router, goalStepIds, handleStepChange]);

  const handleCreateFlow = useCallback(async () => {
    if (!newID.trim() || goalStepIds.length === 0) return;

    const flowId = newID.trim();
    let parsedState: {};
    try {
      parsedState = JSON.parse(initialState);
    } catch {
      parsedState = {};
    }

    addFlow({
      id: flowId,
      status: "pending",
      state: parsedState,
      started_at: new Date().toISOString(),
      plan: previewPlan || undefined,
    });

    setCreating(true);
    router.push(`/flow/${flowId}`);
    setNewID("");
    setGoalStepIds([]);
    setSelectedStep(null);
    setInitialState("{}");
    setShowCreateForm(false);

    try {
      await api.startFlow(flowId, goalStepIds, parsedState);
      await loadFlows();
    } catch (error: any) {
      let errorMessage = "Unknown error";

      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error?.message) {
        errorMessage = error.message;
      }

      removeFlow(flowId);
      toast.error("Failed to create flow: " + errorMessage);
      router.push("/");
    } finally {
      setCreating(false);
    }
  }, [
    newID,
    goalStepIds,
    addFlow,
    router,
    setGoalStepIds,
    setSelectedStep,
    loadFlows,
    removeFlow,
    initialState,
    setShowCreateForm,
    previewPlan,
  ]);

  const value: FlowCreationContextValue = {
    newID,
    setNewID,
    setIDManuallyEdited,
    handleStepChange,
    initialState,
    setInitialState,
    creating,
    handleCreateFlow,
    steps,
    generateID: generateFlowId,
    sortSteps: sortStepsByType,
  };

  return (
    <FlowCreationContext.Provider value={value}>
      {children}
    </FlowCreationContext.Provider>
  );
};
