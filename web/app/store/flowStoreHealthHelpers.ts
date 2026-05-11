import { Step } from "../api";

export interface NodeHealth {
  status: string;
  error?: string;
}

export interface StepHealthInfo extends NodeHealth {
  nodes?: Record<string, NodeHealth>;
}

export const sortNodeIds = (nodeIds: string[]): string[] => {
  return [...nodeIds].sort((a, b) => a.localeCompare(b));
};

const healthRank = (status?: string): number => {
  switch (status) {
    case "unhealthy":
      return 2;
    case "unknown":
      return 1;
    default:
      return 0;
  }
};

const annotateHealthError = (
  nodeId: string,
  error?: string
): string | undefined => {
  return error ? `node ${nodeId}: ${error}` : undefined;
};

const missingHealthError = (_: string): string => {
  return "health not reported";
};

const requiresAllNodeHealth = (step?: Step): boolean => {
  return step?.type !== "flow";
};

const normalizeStepNodes = (
  nodes: Record<string, NodeHealth>,
  nodeIds: string[]
): Record<string, NodeHealth> => {
  const normalized: Record<string, NodeHealth> = {};
  const allNodeIds = new Set([...Object.keys(nodes), ...nodeIds]);

  sortNodeIds(Array.from(allNodeIds)).forEach((nodeId) => {
    normalized[nodeId] = nodes[nodeId] || {
      status: "unknown",
      error: missingHealthError(nodeId),
    };
  });

  return normalized;
};

export const reduceStepHealth = (
  nodes: Record<string, NodeHealth>,
  nodeIds: string[],
  step?: Step
): StepHealthInfo => {
  const normalized = requiresAllNodeHealth(step)
    ? normalizeStepNodes(nodes, nodeIds)
    : normalizeStepNodes(nodes, []);
  const ids = Object.keys(normalized);
  if (ids.length === 0) {
    return { status: "unknown" };
  }

  let status = "healthy";
  let error: string | undefined;

  Object.entries(normalized).forEach(([nodeId, nodeHealth]) => {
    const nextStatus = nodeHealth.status || "unknown";
    if (healthRank(nextStatus) > healthRank(status)) {
      status = nextStatus;
      error = annotateHealthError(nodeId, nodeHealth.error);
      return;
    }
    if (
      healthRank(nextStatus) === healthRank(status) &&
      !error &&
      nodeHealth.error
    ) {
      error = annotateHealthError(nodeId, nodeHealth.error);
    }
  });

  return {
    status,
    ...(error && { error }),
    nodes: normalized,
  };
};

export const toStepHealthMap = (
  healthByNode: Record<string, Record<string, StepHealthInfo>>,
  stepsById: Record<string, Step>
): Record<string, StepHealthInfo> => {
  const byStep: Record<string, Record<string, NodeHealth>> = {};
  const nodeIds = sortNodeIds(Object.keys(healthByNode));

  Object.entries(healthByNode).forEach(([nodeId, stepHealth]) => {
    Object.entries(stepHealth || {}).forEach(([stepId, health]) => {
      if (!byStep[stepId]) {
        byStep[stepId] = {};
      }
      byStep[stepId][nodeId] = {
        status: health.status || "unknown",
        error: health.error,
      };
    });
  });

  return Object.fromEntries(
    Object.entries(byStep).map(([stepId, nodes]) => [
      stepId,
      reduceStepHealth(nodes, nodeIds, stepsById[stepId]),
    ])
  );
};
