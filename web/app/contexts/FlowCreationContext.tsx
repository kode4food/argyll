import React, {
  createContext,
  useContext,
  useState,
  useRef,
  useCallback,
  useEffect,
  useMemo,
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
import { parseState, filterDefaultValues } from "@/utils/stateUtils";
import { generateFlowId, generatePadded } from "@/utils/flowUtils";
import { sortStepsByType } from "@/utils/stepUtils";
import { snapshotFlowPositions } from "@/utils/nodePositioning";
import toast from "react-hot-toast";
import { useT } from "@/app/i18n";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";

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

export const FlowCreationContext =
  createContext<FlowCreationContextValue | null>(null);

export const useFlowCreation = (): FlowCreationContextValue => {
  const ctx = useContext(FlowCreationContext);
  if (!ctx) {
    throw new Error(
      "useFlowCreation must be used within FlowCreationContext.Provider"
    );
  }
  return ctx;
};

export const FlowCreationStateProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const t = useT();
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
      await applyFlowGoalSelectionChange({
        stepIds,
        initialState,
        steps,
        idManuallyEdited,
        setNewID,
        generatePadded,
        setInitialState,
        setGoalSteps,
        updatePreviewPlan,
        clearPreviewPlan,
        getExecutionPlan: (goalStepIds, init) =>
          api.getExecutionPlan(goalStepIds, init),
      });
    },
    [
      initialState,
      idManuallyEdited,
      setNewID,
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

    snapshotFlowPositions(flowId);
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
      let errorMessage = t("flowCreate.unknownError");

      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error?.message) {
        errorMessage = error.message;
      }

      removeFlow(flowId);
      toast.error(t("flowCreate.createFailed", { error: errorMessage }));
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
    t,
  ]);

  const value = useMemo<FlowCreationContextValue>(
    () => ({
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
    }),
    [
      newID,
      setIDManuallyEdited,
      handleStepChange,
      initialState,
      creating,
      handleCreateFlow,
      steps,
    ]
  );

  return (
    <FlowCreationContext.Provider value={value}>
      {children}
    </FlowCreationContext.Provider>
  );
};
