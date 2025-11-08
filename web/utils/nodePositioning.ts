import { Node } from "@xyflow/react";

const STORAGE_KEY = "spuds-step-positions";

export const saveNodePositions = (nodes: Node[]) => {
  const positions: Record<string, { x: number; y: number }> = {};
  nodes.forEach((node) => {
    positions[node.id] = { x: node.position.x, y: node.position.y };
  });
  localStorage.setItem(STORAGE_KEY, JSON.stringify(positions));
};

export const loadNodePositions = (): Record<
  string,
  { x: number; y: number }
> => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : {};
  } catch {
    return {};
  }
};
