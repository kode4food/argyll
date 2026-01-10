import { Node } from "@xyflow/react";

export type NodePositionScope =
  | { type: "overview" }
  | { type: "flow"; flowId: string };

export type NodePositions = Record<string, { x: number; y: number }>;

const STORAGE_PREFIX = "argyll-step-positions";
export const OVERVIEW_STORAGE_KEY = STORAGE_PREFIX;

export const getFlowStorageKey = (flowId: string) =>
  `${STORAGE_PREFIX}:flow:${flowId}`;

const resolveStorageKey = (scope?: NodePositionScope): string => {
  if (scope?.type === "flow") {
    return getFlowStorageKey(scope.flowId);
  }
  return OVERVIEW_STORAGE_KEY;
};

export const saveNodePositionsMap = (
  positions: NodePositions,
  scope?: NodePositionScope
) => {
  localStorage.setItem(resolveStorageKey(scope), JSON.stringify(positions));
};

export const saveNodePositions = (nodes: Node[], scope?: NodePositionScope) => {
  const positions: NodePositions = {};
  nodes.forEach((node) => {
    positions[node.id] = { x: node.position.x, y: node.position.y };
  });
  saveNodePositionsMap(positions, scope);
};

export const loadNodePositions = (scope?: NodePositionScope): NodePositions => {
  try {
    const stored = localStorage.getItem(resolveStorageKey(scope));
    return stored ? JSON.parse(stored) : {};
  } catch {
    return {};
  }
};

export const snapshotFlowPositions = (flowId: string): void => {
  if (!flowId) {
    return;
  }
  const positions = loadNodePositions();
  saveNodePositionsMap(positions, { type: "flow", flowId });
};
