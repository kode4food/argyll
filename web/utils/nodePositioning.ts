import { Node } from "@xyflow/react";

export type NodePositionScope =
  | { type: "overview" }
  | { type: "flow"; flowId: string }
  | { type: "space"; spaceId: string };

export type NodePositions = Record<string, { x: number; y: number }>;

const STORAGE_PREFIX = "argyll-step-positions";
export const OVERVIEW_STORAGE_KEY = STORAGE_PREFIX;

export const getFlowStorageKey = (flowId: string) =>
  `${STORAGE_PREFIX}:flow:${flowId}`;

const getSpaceStorageKey = (spaceId: string) =>
  `${STORAGE_PREFIX}:space:${spaceId}`;

const resolveStorageKey = (scope?: NodePositionScope): string => {
  if (scope?.type === "flow") {
    return getFlowStorageKey(scope.flowId);
  }
  if (scope?.type === "space") {
    return getSpaceStorageKey(scope.spaceId);
  }
  return OVERVIEW_STORAGE_KEY;
};

export const saveNodePositionsMap = (
  positions: NodePositions,
  scope?: NodePositionScope
) => {
  const existing = readPositions(resolveStorageKey(scope));
  const merged = { ...existing, ...positions };
  localStorage.setItem(resolveStorageKey(scope), JSON.stringify(merged));
};

export const saveNodePositions = (nodes: Node[], scope?: NodePositionScope) => {
  const positions: NodePositions = {};
  nodes.forEach((node) => {
    positions[node.id] = { x: node.position.x, y: node.position.y };
  });
  saveNodePositionsMap(positions, scope);
};

const readPositions = (key: string): NodePositions => {
  try {
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : {};
  } catch {
    return {};
  }
};

export const loadNodePositions = (scope?: NodePositionScope): NodePositions => {
  const positions = readPositions(resolveStorageKey(scope));
  return scope?.type === "space"
    ? { ...readPositions(OVERVIEW_STORAGE_KEY), ...positions }
    : positions;
};

export const snapshotFlowPositions = (
  flowId: string,
  origin?: NodePositionScope
): void => {
  if (!flowId) {
    return;
  }
  const positions = loadNodePositions(origin);
  saveNodePositionsMap(positions, { type: "flow", flowId });
};
