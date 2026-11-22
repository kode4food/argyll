import { useEffect } from "react";
import { useUI } from "@/app/contexts/UIContext";
import {
  useExecutions,
  useFlowData,
  useFlowLoading,
  useFlowNotFound,
  useResolvedAttributes,
  useSelectedFlow,
  useSteps,
} from "@/app/store/flowStore";
import { isValidTimestamp } from "@/utils/dates";

export const useLiveDiagram = () => {
  const selectedFlow = useSelectedFlow();
  const flowData = useFlowData();
  const executions = useExecutions();
  const resolved = useResolvedAttributes();
  const loading = useFlowLoading();
  const flowNotFound = useFlowNotFound();
  const steps = useSteps();
  const { clearPreviewPlan, setGoalSteps } = useUI();

  useEffect(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [clearPreviewPlan, setGoalSteps]);

  const notFoundId = flowNotFound && !loading ? selectedFlow : null;
  const startedAt =
    flowData && isValidTimestamp(flowData.started_at)
      ? new Date(flowData.started_at).toLocaleString()
      : null;
  const endedAt =
    flowData?.completed_at && isValidTimestamp(flowData.completed_at)
      ? new Date(flowData.completed_at).toLocaleString()
      : null;

  return {
    flowData,
    executions,
    resolved,
    steps,
    notFoundId,
    startedAt,
    endedAt,
  };
};
