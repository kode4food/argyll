import { useMemo } from "react";
import { Step, HealthStatus, StepHealth } from "../api";
import { useWorkflowStore } from "../store/workflowStore";

export const useStepHealth = (step: Step): StepHealth => {
  const healthInfo = useWorkflowStore((state) => state.stepHealth[step.id]);
  return useMemo(() => {
    // For HTTP steps, check if health check is configured
    if (
      (step.type === "sync" || step.type === "async") &&
      !step.http?.health_check
    ) {
      return { status: "unconfigured" };
    }

    // Use health from store
    const status: HealthStatus =
      (healthInfo?.status as HealthStatus) || "unknown";

    return {
      status,
      error: healthInfo?.error,
    };
  }, [step.type, step.http?.health_check, healthInfo]);
};
