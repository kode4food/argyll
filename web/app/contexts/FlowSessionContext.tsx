import React, { createContext, useContext, useMemo, useEffect } from "react";
import {
  useSelectedFlow,
  useLoadFlows,
  useLoadSteps,
  useSteps,
  useFlows,
  useUpdateFlowStatus,
  useFlowData,
  useFlowLoading,
  useFlowNotFound,
  useIsFlowMode,
  useExecutions,
  useResolvedAttributes,
  useFlowStore,
  useFlowError,
} from "../store/flowStore";

type FlowSessionValue = {
  selectedFlow: string | null;
  selectFlow: (flowId: string | null) => void;
  loadFlows: () => Promise<void>;
  loadSteps: () => Promise<void>;
  steps: ReturnType<typeof useSteps>;
  flows: ReturnType<typeof useFlows>;
  updateFlowStatus: ReturnType<typeof useUpdateFlowStatus>;
  flowData: ReturnType<typeof useFlowData>;
  loading: boolean;
  flowNotFound: boolean;
  isFlowMode: boolean;
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
  const loadSteps = useLoadSteps();
  const steps = useSteps();
  const flows = useFlows();
  const updateFlowStatus = useUpdateFlowStatus();
  const flowData = useFlowData();
  const loading = useFlowLoading();
  const flowNotFound = useFlowNotFound();
  const isFlowMode = useIsFlowMode();
  const executions = useExecutions();
  const resolvedAttributes = useResolvedAttributes();
  const flowError = useFlowError();

  useEffect(() => {
    // Steps are loaded via engine WebSocket subscribe_state
    // Flows list still needs HTTP API since engine only tracks active flows
    loadFlows?.();
  }, [loadFlows]);

  const value = useMemo(
    () => ({
      selectedFlow,
      selectFlow,
      loadFlows,
      loadSteps,
      steps,
      flows,
      updateFlowStatus,
      flowData,
      loading,
      flowNotFound,
      isFlowMode,
      executions,
      resolvedAttributes,
      flowError,
    }),
    [
      selectedFlow,
      selectFlow,
      loadFlows,
      loadSteps,
      steps,
      flows,
      updateFlowStatus,
      flowData,
      loading,
      flowNotFound,
      isFlowMode,
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
