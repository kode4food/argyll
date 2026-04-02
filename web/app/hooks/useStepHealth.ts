import { useMemo } from "react";
import { Step, HealthStatus, NodeStepHealth, StepHealth } from "../api";
import { useFlowStore } from "../store/flowStore";

const sortedNodes = (
  perNode: Record<string, { status: string; error?: string }>
): NodeStepHealth[] =>
  Object.entries(perNode)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([nodeId, h]) => ({
      nodeId,
      status: h.status as HealthStatus,
      error: h.error,
    }));

export const useStepHealth = (
  step: Step
): StepHealth & { nodes?: NodeStepHealth[] } => {
  const healthInfo = useFlowStore((state) => state.stepHealth[step.id]);

  return useMemo(() => {
    if (
      (step.type === "sync" || step.type === "async") &&
      !step.http?.health_check
    ) {
      return { status: "unconfigured" };
    }

    const status = (healthInfo?.status as HealthStatus) || "unknown";
    const nodes = sortedNodes(healthInfo?.nodes ?? {});
    return {
      status,
      error: healthInfo?.error,
      ...(nodes.length > 0 && { nodes }),
    };
  }, [step.type, step.http?.health_check, healthInfo]);
};
