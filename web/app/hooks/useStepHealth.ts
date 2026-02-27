import { useMemo } from "react";
import { Step, HealthStatus, StepHealth } from "../api";
import { useFlowStore } from "../store/flowStore";

export const useStepHealth = (step: Step): StepHealth => {
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

    return {
      status,
      error: healthInfo?.error,
    };
  }, [step.type, step.http?.health_check, healthInfo]);
};
