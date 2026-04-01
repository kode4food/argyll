import { useMemo } from "react";
import { Step, HealthStatus, NodeStepHealth, StepHealth } from "../api";
import { useFlowStore } from "../store/flowStore";

export const useStepHealth = (
  step: Step
): StepHealth & { nodes?: NodeStepHealth[] } => {
  const healthInfo = useFlowStore((state) => state.stepHealth[step.id]);

  return useMemo(() => {
    // For HTTP steps, check if health check is configured
    if (
      (step.type === "sync" || step.type === "async") &&
      !step.http?.health_check
    ) {
      return { status: "unconfigured" };
    }

    const status: HealthStatus =
      (healthInfo?.status as HealthStatus) || "unknown";
    const nodes = Object.entries(healthInfo?.nodes || {})
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([nodeId, nodeHealth]) => ({
        nodeId,
        status: (nodeHealth.status as HealthStatus) || "unknown",
        error: nodeHealth.error,
      }));

    return {
      status,
      error: healthInfo?.error,
      ...(nodes.length > 0 && { nodes }),
    };
  }, [step.type, step.http?.health_check, healthInfo]);
};
