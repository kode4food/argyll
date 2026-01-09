import { Node } from "@xyflow/react";
import {
  groupNodesByLevel,
  calculateNodeLevel,
  findClosestStepInLevel,
  findNextStepInDirection,
  LevelStep,
} from "./navigationUtils";

describe("navigationUtils", () => {
  describe("groupNodesByLevel", () => {
    test("groups nodes by x position into levels", () => {
      const nodes: Node[] = [
        { id: "a", position: { x: 0, y: 100 }, data: {} },
        { id: "b", position: { x: 400, y: 200 }, data: {} },
        { id: "c", position: { x: 50, y: 150 }, data: {} },
        { id: "d", position: { x: 800, y: 100 }, data: {} },
      ];

      const levels = groupNodesByLevel(nodes);

      expect(levels.size).toBe(3);
      expect(levels.get(0)).toHaveLength(2);
      expect(levels.get(1)).toHaveLength(1);
      expect(levels.get(2)).toHaveLength(1);
    });

    test("sorts steps within each level by y position", () => {
      const nodes: Node[] = [
        { id: "a", position: { x: 0, y: 300 }, data: {} },
        { id: "b", position: { x: 50, y: 100 }, data: {} },
        { id: "c", position: { x: 25, y: 200 }, data: {} },
      ];

      const levels = groupNodesByLevel(nodes);
      const level0 = levels.get(0);

      expect(level0?.[0].id).toBe("b");
      expect(level0?.[1].id).toBe("c");
      expect(level0?.[2].id).toBe("a");
    });

    test("handles custom level width", () => {
      const nodes: Node[] = [
        { id: "a", position: { x: 0, y: 100 }, data: {} },
        { id: "b", position: { x: 100, y: 200 }, data: {} },
        { id: "c", position: { x: 200, y: 150 }, data: {} },
      ];

      const levels = groupNodesByLevel(nodes, 100);

      expect(levels.size).toBe(3);
      expect(levels.get(0)).toHaveLength(1);
      expect(levels.get(1)).toHaveLength(1);
      expect(levels.get(2)).toHaveLength(1);
    });

    test("handles empty nodes array", () => {
      const levels = groupNodesByLevel([]);
      expect(levels.size).toBe(0);
    });
  });

  describe("calculateNodeLevel", () => {
    test("calculates correct level for x position", () => {
      expect(calculateNodeLevel(0)).toBe(0);
      expect(calculateNodeLevel(200)).toBe(1);
      expect(calculateNodeLevel(400)).toBe(1);
      expect(calculateNodeLevel(600)).toBe(2);
      expect(calculateNodeLevel(800)).toBe(2);
    });

    test("handles custom level width", () => {
      expect(calculateNodeLevel(0, 100)).toBe(0);
      expect(calculateNodeLevel(50, 100)).toBe(1);
      expect(calculateNodeLevel(100, 100)).toBe(1);
      expect(calculateNodeLevel(150, 100)).toBe(2);
    });
  });

  describe("findClosestStepInLevel", () => {
    test("finds closest step by y position", () => {
      const steps: LevelStep[] = [
        { id: "a", y: 100 },
        { id: "b", y: 200 },
        { id: "c", y: 300 },
      ];

      expect(findClosestStepInLevel(150, steps)).toBe("a");
      expect(findClosestStepInLevel(90, steps)).toBe("a");
      expect(findClosestStepInLevel(320, steps)).toBe("c");
      expect(findClosestStepInLevel(250, steps)).toBe("b");
    });

    test("handles exact match", () => {
      const steps: LevelStep[] = [
        { id: "a", y: 100 },
        { id: "b", y: 200 },
      ];

      expect(findClosestStepInLevel(100, steps)).toBe("a");
      expect(findClosestStepInLevel(200, steps)).toBe("b");
    });
  });

  describe("findNextStepInDirection", () => {
    const nodes: Node[] = [
      { id: "a", position: { x: 0, y: 100 }, data: {} },
      { id: "b", position: { x: 0, y: 200 }, data: {} },
      { id: "c", position: { x: 400, y: 150 }, data: {} },
      { id: "d", position: { x: 400, y: 250 }, data: {} },
      { id: "e", position: { x: 800, y: 200 }, data: {} },
    ];

    const stepsByLevel = groupNodesByLevel(nodes);

    test("returns first step when no current step", () => {
      const next = findNextStepInDirection("up", null, nodes, stepsByLevel);
      expect(next).toBe("a");
    });

    test("navigates up within same level", () => {
      const next = findNextStepInDirection("up", "b", nodes, stepsByLevel);
      expect(next).toBe("a");
    });

    test("returns null when at top of level", () => {
      const next = findNextStepInDirection("up", "a", nodes, stepsByLevel);
      expect(next).toBeNull();
    });

    test("navigates down within same level", () => {
      const next = findNextStepInDirection("down", "a", nodes, stepsByLevel);
      expect(next).toBe("b");
    });

    test("returns null when at bottom of level", () => {
      const next = findNextStepInDirection("down", "b", nodes, stepsByLevel);
      expect(next).toBeNull();
    });

    test("navigates left to previous level", () => {
      const next = findNextStepInDirection("left", "c", nodes, stepsByLevel);
      expect(next).toBe("a");
    });

    test("returns null when at leftmost level", () => {
      const next = findNextStepInDirection("left", "a", nodes, stepsByLevel);
      expect(next).toBeNull();
    });

    test("navigates right to next level", () => {
      const next = findNextStepInDirection("right", "a", nodes, stepsByLevel);
      expect(next).toBe("c");
    });

    test("returns null when at rightmost level", () => {
      const next = findNextStepInDirection("right", "e", nodes, stepsByLevel);
      expect(next).toBeNull();
    });

    test("finds closest step by y position when moving left", () => {
      const next = findNextStepInDirection("left", "d", nodes, stepsByLevel);
      expect(next).toBe("b");
    });

    test("finds closest step by y position when moving right", () => {
      const next = findNextStepInDirection("right", "b", nodes, stepsByLevel);
      expect(next).toBe("c");
    });

    test("returns null for invalid direction", () => {
      const next = findNextStepInDirection(
        "invalid" as any,
        "a",
        nodes,
        stepsByLevel
      );
      expect(next).toBeNull();
    });

    test("returns null when current step not found", () => {
      const next = findNextStepInDirection(
        "up",
        "missing",
        nodes,
        stepsByLevel
      );
      expect(next).toBeNull();
    });

    test("handles custom level width", () => {
      const customNodes: Node[] = [
        { id: "a", position: { x: 0, y: 100 }, data: {} },
        { id: "b", position: { x: 100, y: 100 }, data: {} },
      ];
      const customLevels = groupNodesByLevel(customNodes, 100);

      const next = findNextStepInDirection(
        "right",
        "a",
        customNodes,
        customLevels,
        100
      );
      expect(next).toBe("b");
    });
  });
});
