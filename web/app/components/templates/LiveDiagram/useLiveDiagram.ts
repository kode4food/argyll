import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useUI } from "@/app/contexts/UIContext";
import { useChatAction } from "@/app/hooks/useChatAction";
import { useChatStore } from "@/app/store/chatStore";
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
  const navigate = useNavigate();
  const chatAction = useChatStore((state) => state.action);

  useEffect(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [clearPreviewPlan, setGoalSteps]);

  useChatAction({
    open_flow: ({ flow_id }) => {
      if (flow_id !== selectedFlow) navigate(`/flow/${flow_id}`);
    },
  });

  // The overview runs every other chat action once it opens
  useEffect(() => {
    if (chatAction && chatAction.type !== "open_flow") navigate("/");
  }, [chatAction, navigate]);

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
