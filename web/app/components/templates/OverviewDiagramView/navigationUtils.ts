import { Node } from "@xyflow/react";

export interface LevelStep {
  id: string;
  y: number;
}

export function groupNodesByLevel(
  nodes: Node[],
  levelWidth: number = 400
): Map<number, LevelStep[]> {
  const levels = new Map<number, LevelStep[]>();

  nodes.forEach((node) => {
    const level = Math.round(node.position.x / levelWidth);
    if (!levels.has(level)) {
      levels.set(level, []);
    }
    levels.get(level)!.push({ id: node.id, y: node.position.y });
  });

  // Sort steps within each level by Y position
  levels.forEach((steps) => {
    steps.sort((a, b) => a.y - b.y);
  });

  return levels;
}

export function calculateNodeLevel(
  xPosition: number,
  levelWidth: number = 400
): number {
  return Math.round(xPosition / levelWidth);
}

export function findClosestStepInLevel(
  currentY: number,
  levelSteps: LevelStep[]
): string {
  return levelSteps.reduce((prev, curr) => {
    const prevDist = Math.abs(prev.y - currentY);
    const currDist = Math.abs(curr.y - currentY);
    return currDist < prevDist ? curr : prev;
  }).id;
}

export function findNextStepInDirection(
  direction: "up" | "down" | "left" | "right",
  currentStepId: string | null,
  nodes: Node[],
  stepsByLevel: Map<number, LevelStep[]>,
  levelWidth: number = 400
): string | null {
  // If no current step, return first step in first level
  if (!currentStepId) {
    const firstLevel = Math.min(...Array.from(stepsByLevel.keys()));
    const stepsInLevel = stepsByLevel.get(firstLevel);
    return stepsInLevel?.[0]?.id || null;
  }

  const currentNode = nodes.find((n) => n.id === currentStepId);
  if (!currentNode) return null;

  const currentLevel = calculateNodeLevel(currentNode.position.x, levelWidth);
  const currentLevelSteps = stepsByLevel.get(currentLevel) || [];
  const currentIndex = currentLevelSteps.findIndex(
    (s) => s.id === currentStepId
  );

  switch (direction) {
    case "up": {
      // Move up within same level
      if (currentIndex > 0) {
        return currentLevelSteps[currentIndex - 1].id;
      }
      return null;
    }
    case "down": {
      // Move down within same level
      if (currentIndex < currentLevelSteps.length - 1) {
        return currentLevelSteps[currentIndex + 1].id;
      }
      return null;
    }
    case "left": {
      // Move to previous level, closest step by Y position
      const prevLevel = currentLevel - 1;
      const prevLevelSteps = stepsByLevel.get(prevLevel);
      if (!prevLevelSteps || prevLevelSteps.length === 0) return null;

      return findClosestStepInLevel(currentNode.position.y, prevLevelSteps);
    }
    case "right": {
      // Move to next level, closest step by Y position
      const nextLevel = currentLevel + 1;
      const nextLevelSteps = stepsByLevel.get(nextLevel);
      if (!nextLevelSteps || nextLevelSteps.length === 0) return null;

      return findClosestStepInLevel(currentNode.position.y, nextLevelSteps);
    }
    default:
      return null;
  }
}
