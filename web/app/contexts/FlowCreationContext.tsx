import React, {
  createContext,
  useContext,
  useState,
  useRef,
  useCallback,
  useEffect,
} from "react";
import { useNavigate } from "react-router-dom";
import { Step } from "../api";
import {
  useSteps,
  useLoadFlows,
  useAddFlow,
  useRemoveFlow,
} from "../store/flowStore";
import { useUI } from "../contexts/UIContext";
import { useThrottledValue } from "./useThrottledValue";
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
  const navigate = useNavigate();
  const steps = useSteps();
  const loadFlows = useLoadFlows();
  const addFlow = useAddFlow();
  const removeFlow = useRemoveFlow();
  const {
    previewPlan,
    updatePreviewPlan,
    clearPreviewPlan,
    goalSteps,
    setGoalSteps,
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
    setGoalSteps([]);
    setInitialState("{}");
    setIDManuallyEdited(false);
    clearPreviewPlan();
    initializedGoalsRef.current = false;
  }, [clearPreviewPlan, setGoalSteps]);

  const handleStepChange = useCallback(
    async (stepIds: string[]) => {
      const currentState = parseState(initialState);
      const nonDefaultState = filterDefaultValues(currentState, steps);

      if (stepIds.length === 0) {
        setInitialState(JSON.stringify(nonDefaultState, null, 2));
        clearPreviewPlan();
        setGoalSteps([]);
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
              setGoalSteps(finalGoals);
              await updatePreviewPlan(finalGoals, stateWithDefaults);
              return;
            }
          } catch {}
        }

        setGoalSteps(stepIds);
        await updatePreviewPlan(stepIds, stateWithDefaults);
      } catch (error) {
        clearPreviewPlan();
        setGoalSteps(stepIds);
      }
    },
    [
      initialState,
      idManuallyEdited,
      steps,
      setGoalSteps,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  const throttledInitialState = useThrottledValue(initialState, 500);

  useEffect(() => {
    if (!showCreateForm || goalSteps.length === 0) {
      return;
    }

    const currentState = parseState(throttledInitialState);
    const nonDefaultState = filterDefaultValues(currentState, steps);

    if (Object.keys(currentState).length >= 0) {
      updatePreviewPlan(goalSteps, nonDefaultState).catch(() => {});
    }
  }, [
    throttledInitialState,
    showCreateForm,
    goalSteps,
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

    if (goalSteps.length === 0) {
      initializedGoalsRef.current = false;
      return;
    }

    if (!initializedGoalsRef.current) {
      initializedGoalsRef.current = true;
      handleStepChange(goalSteps);
    }
  }, [showCreateForm, goalSteps, handleStepChange]);

  const handleCreateFlow = useCallback(async () => {
    if (!newID.trim() || goalSteps.length === 0) return;

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
    setNewID("");
    setGoalSteps([]);
    setInitialState("{}");
    setShowCreateForm(false);

    try {
      await api.startFlow(flowId, goalSteps, parsedState);
      await loadFlows();
      navigate(`/flow/${flowId}`);
    } catch (error: any) {
      let errorMessage = "Unknown error";

      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error?.message) {
        errorMessage = error.message;
      }

      removeFlow(flowId);
      toast.error("Failed to create flow: " + errorMessage);
      navigate("/");
    } finally {
      setCreating(false);
    }
  }, [
    newID,
    goalSteps,
    addFlow,
    navigate,
    setGoalSteps,
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
