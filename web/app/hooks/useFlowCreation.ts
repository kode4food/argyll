import React, { useState, useRef, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Step } from "../api";
import {
  useSteps,
  useLoadFlows,
  useAddFlow,
  useRemoveFlow,
} from "../store/flowStore";
import { useUI } from "../contexts/UIContext";
import { useThrottledValue } from "@/app/contexts/useThrottledValue";
import { api } from "../api";
import { parseState, filterDefaultValues } from "@/utils/stateUtils";
import { snapshotFlowPositions } from "@/utils/nodePositioning";
import toast from "react-hot-toast";
import { useT } from "@/app/i18n";
import { applyFlowGoalSelectionChange } from "@/utils/flowGoalSelectionModel";

export const useFlowCreation = () => {
  const t = useT();
  const navigate = useNavigate();
  const steps = useSteps();
  const loadFlows = useLoadFlows();
  const addFlow = useAddFlow();
  const removeFlow = useRemoveFlow();
  const {
    setPreviewPlan,
    updatePreviewPlan,
    clearPreviewPlan,
    goalSteps,
    setGoalSteps,
  } = useUI();

  const [newID, setNewID] = useState("");
  const [initialState, setInitialState] = useState("{}");
  const [compensate, setCompensate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [idManuallyEdited, setIDManuallyEdited] = useState(false);
  const initializedGoalsRef = useRef(false);

  const resetForm = useCallback(() => {
    setNewID("");
    setGoalSteps([]);
    setInitialState("{}");
    setCompensate(false);
    setIDManuallyEdited(false);
    clearPreviewPlan();
    initializedGoalsRef.current = false;
  }, [clearPreviewPlan, setGoalSteps]);

  const handleStepChange = useCallback(
    async (stepIds: string[]) => {
      initializedGoalsRef.current = true;
      await applyFlowGoalSelectionChange({
        stepIds,
        initialState,
        steps,
        idManuallyEdited,
        setNewID,
        setInitialState,
        setGoalSteps,
        updatePreviewPlan,
        setPreviewPlan,
        clearPreviewPlan,
      });
    },
    [
      initialState,
      idManuallyEdited,
      setNewID,
      steps,
      setGoalSteps,
      setPreviewPlan,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  const throttledInitialState = useThrottledValue(initialState, 500);

  useEffect(() => {
    if (goalSteps.length === 0) {
      return;
    }

    const currentState = parseState(throttledInitialState);
    const nonDefaultState = filterDefaultValues(currentState, steps);

    updatePreviewPlan(goalSteps, nonDefaultState).catch(() => {});
  }, [throttledInitialState, goalSteps, steps, updatePreviewPlan]);

  useEffect(() => {
    if (goalSteps.length === 0) {
      return;
    }

    if (!initializedGoalsRef.current) {
      initializedGoalsRef.current = true;
      handleStepChange(goalSteps);
    }
  }, [goalSteps, handleStepChange]);

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
      timestamp: new Date().toISOString(),
    });

    setCreating(true);

    try {
      await api.startFlow({
        id: flowId,
        goalSteps,
        initialState: parsedState,
        compensate,
      });
      await loadFlows();
      resetForm();
      navigate(`/flow/${flowId}`);
    } catch (error: any) {
      const errorMessage = error?.message || t("flowCreate.unknownError");

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
    loadFlows,
    removeFlow,
    initialState,
    compensate,
    resetForm,
    t,
  ]);

  return {
    newID,
    setNewID,
    setIDManuallyEdited,
    handleStepChange,
    initialState,
    setInitialState,
    compensate,
    setCompensate,
    creating,
    handleCreateFlow,
    steps,
  };
};
