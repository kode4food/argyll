import { useMemo } from "react";
import { useShallow } from "zustand/react/shallow";
import { Step, HealthStatus, NodeStepHealth, StepHealth } from "../api";
import { useFlowStore } from "../store/flowStore";

const healthRank = (status: string): number => {
  if (status === "unhealthy") return 2;
  if (status === "unknown") return 1;
  return 0;
};

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
  const goalHealths = useFlowStore(
    useShallow((state) => {
      if (step.type !== "flow" || !step.flow?.goals?.length) return null;
      return step.flow.goals.map((id) => state.stepHealth[id] ?? null);
    })
  );

  return useMemo(() => {
    // Script steps run only on the leader; no meaningful per-node breakdown
    if (step.type === "script") {
      return {
        status: (healthInfo?.status as HealthStatus) || "unknown",
        error: healthInfo?.error,
      };
    }

    // HTTP steps without a health check configured
    if (
      (step.type === "sync" || step.type === "async") &&
      !step.http?.health_check
    ) {
      return { status: "unconfigured" };
    }

    // Flow steps: derive overall status and per-node health from goal steps
    if (step.type === "flow" && goalHealths) {
      const perNode: Record<string, { status: string; error?: string }> = {};
      let status: HealthStatus | null = null;
      let error: string | undefined;

      for (const goalHealth of goalHealths) {
        if (!goalHealth) continue;
        const s = (goalHealth.status as HealthStatus) || "unknown";
        if (status === null || healthRank(s) > healthRank(status)) {
          status = s;
          error = goalHealth.error;
        }
        for (const [nodeId, h] of Object.entries(goalHealth.nodes ?? {})) {
          const nextStatus = h.status || "unknown";
          const existing = perNode[nodeId];
          if (
            !existing ||
            healthRank(nextStatus) > healthRank(existing.status)
          ) {
            perNode[nodeId] = { status: nextStatus, error: h.error };
          }
        }
      }

      const nodes = sortedNodes(perNode);
      return {
        status: status ?? "unknown",
        error,
        ...(nodes.length > 0 && { nodes }),
      };
    }

    // Sync/Async: per-node health from store
    const status = (healthInfo?.status as HealthStatus) || "unknown";
    const nodes = sortedNodes(healthInfo?.nodes ?? {});
    return {
      status,
      error: healthInfo?.error,
      ...(nodes.length > 0 && { nodes }),
    };
  }, [
    step.type,
    step.http?.health_check,
    step.flow?.goals,
    healthInfo,
    goalHealths,
  ]);
};
