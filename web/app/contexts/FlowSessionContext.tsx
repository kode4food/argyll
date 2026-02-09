import React, { createContext, useContext, useMemo, useEffect } from "react";
import {
  useSelectedFlow,
  useLoadFlows,
  useLoadMoreFlows,
  useLoadSteps,
  useSteps,
  useFlows,
  useFlowsHasMore,
  useFlowsLoading,
  useUpdateFlowStatus,
  useFlowData,
  useFlowLoading,
  useFlowNotFound,
  useExecutions,
  useResolvedAttributes,
  useFlowStore,
  useFlowError,
} from "../store/flowStore";

type FlowSessionValue = {
  selectedFlow: string | null;
  selectFlow: (flowId: string | null) => void;
  loadFlows: () => Promise<void>;
  loadMoreFlows: () => Promise<void>;
  loadSteps: () => Promise<void>;
  steps: ReturnType<typeof useSteps>;
  flows: ReturnType<typeof useFlows>;
  flowsHasMore: boolean;
  flowsLoading: boolean;
  updateFlowStatus: ReturnType<typeof useUpdateFlowStatus>;
  flowData: ReturnType<typeof useFlowData>;
  loading: boolean;
  flowNotFound: boolean;
  executions: ReturnType<typeof useExecutions>;
  resolvedAttributes: ReturnType<typeof useResolvedAttributes>;
  flowError: string | null;
};

const FlowSessionContext = createContext<FlowSessionValue | null>(null);

export const FlowSessionProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const selectedFlow = useSelectedFlow();
  const selectFlow = useFlowStore((s) => s.selectFlow);
  const loadFlows = useLoadFlows();
  const loadMoreFlows = useLoadMoreFlows();
  const loadSteps = useLoadSteps();
  const steps = useSteps();
  const flows = useFlows();
  const flowsHasMore = useFlowsHasMore();
  const flowsLoading = useFlowsLoading();
  const updateFlowStatus = useUpdateFlowStatus();
  const flowData = useFlowData();
  const loading = useFlowLoading();
  const flowNotFound = useFlowNotFound();
  const executions = useExecutions();
  const resolvedAttributes = useResolvedAttributes();
  const flowError = useFlowError();

  useEffect(() => {
    loadFlows?.();
  }, [loadFlows]);

  const value = useMemo(
    () => ({
      selectedFlow,
      selectFlow,
      loadFlows,
      loadMoreFlows,
      loadSteps,
      steps,
      flows,
      flowsHasMore,
      flowsLoading,
      updateFlowStatus,
      flowData,
      loading,
      flowNotFound,
      executions,
      resolvedAttributes,
      flowError,
    }),
    [
      selectedFlow,
      selectFlow,
      loadFlows,
      loadMoreFlows,
      loadSteps,
      steps,
      flows,
      flowsHasMore,
      flowsLoading,
      updateFlowStatus,
      flowData,
      loading,
      flowNotFound,
      executions,
      resolvedAttributes,
      flowError,
    ]
  );

  return (
    <FlowSessionContext.Provider value={value}>
      {children}
    </FlowSessionContext.Provider>
  );
};

export const useFlowSession = (): FlowSessionValue => {
  const ctx = useContext(FlowSessionContext);
  if (!ctx) {
    throw new Error("useFlowSession must be used within FlowSessionProvider");
  }
  return ctx;
};
