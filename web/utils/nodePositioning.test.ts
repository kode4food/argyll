import {
  saveNodePositions,
  loadNodePositions,
  OVERVIEW_STORAGE_KEY,
  getFlowStorageKey,
  snapshotFlowPositions,
} from "./nodePositioning";
import { Node } from "@xyflow/react";

describe("nodePositioning", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  describe("saveNodePositions", () => {
    test("saves node positions to localStorage", () => {
      const nodes: Node[] = [
        {
          id: "node-1",
          position: { x: 100, y: 200 },
          data: {},
        },
        {
          id: "node-2",
          position: { x: 300, y: 400 },
          data: {},
        },
      ];

      saveNodePositions(nodes);

      const stored = localStorage.getItem(OVERVIEW_STORAGE_KEY);
      expect(stored).toBeTruthy();
      const parsed = JSON.parse(stored!);
      expect(parsed).toEqual({
        "node-1": { x: 100, y: 200 },
        "node-2": { x: 300, y: 400 },
      });
    });

    test("handles empty node array", () => {
      saveNodePositions([]);

      const stored = localStorage.getItem(OVERVIEW_STORAGE_KEY);
      expect(stored).toBe("{}");
    });

    test("overwrites existing positions", () => {
      const nodes1: Node[] = [
        { id: "node-1", position: { x: 100, y: 200 }, data: {} },
      ];
      const nodes2: Node[] = [
        { id: "node-1", position: { x: 500, y: 600 }, data: {} },
      ];

      saveNodePositions(nodes1);
      saveNodePositions(nodes2);

      const stored = localStorage.getItem(OVERVIEW_STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["node-1"]).toEqual({ x: 500, y: 600 });
    });

    test("handles nodes with fractional positions", () => {
      const nodes: Node[] = [
        { id: "node-1", position: { x: 123.456, y: 789.012 }, data: {} },
      ];

      saveNodePositions(nodes);

      const stored = localStorage.getItem(OVERVIEW_STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["node-1"]).toEqual({ x: 123.456, y: 789.012 });
    });

    test("handles nodes with negative positions", () => {
      const nodes: Node[] = [
        { id: "node-1", position: { x: -100, y: -200 }, data: {} },
      ];

      saveNodePositions(nodes);

      const stored = localStorage.getItem(OVERVIEW_STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["node-1"]).toEqual({ x: -100, y: -200 });
    });
  });

  describe("loadNodePositions", () => {
    test("loads node positions from localStorage", () => {
      const positions = {
        "node-1": { x: 100, y: 200 },
        "node-2": { x: 300, y: 400 },
      };
      localStorage.setItem(OVERVIEW_STORAGE_KEY, JSON.stringify(positions));

      const result = loadNodePositions();

      expect(result).toEqual(positions);
    });

    test("returns empty object when no data stored", () => {
      const result = loadNodePositions();

      expect(result).toEqual({});
    });

    test("returns empty object on parse error", () => {
      localStorage.setItem(OVERVIEW_STORAGE_KEY, "invalid json");

      const result = loadNodePositions();

      expect(result).toEqual({});
    });

    test("handles empty stored object", () => {
      localStorage.setItem(OVERVIEW_STORAGE_KEY, "{}");

      const result = loadNodePositions();

      expect(result).toEqual({});
    });

    test("preserves position data types", () => {
      const positions = {
        "node-1": { x: 123.456, y: -789.012 },
      };
      localStorage.setItem(OVERVIEW_STORAGE_KEY, JSON.stringify(positions));

      const result = loadNodePositions();

      expect(result["node-1"].x).toBe(123.456);
      expect(result["node-1"].y).toBe(-789.012);
    });
  });

  describe("round trip", () => {
    test("saves and loads positions correctly", () => {
      const nodes: Node[] = [
        { id: "node-1", position: { x: 100, y: 200 }, data: {} },
        { id: "node-2", position: { x: 300, y: 400 }, data: {} },
      ];

      saveNodePositions(nodes);
      const loaded = loadNodePositions();

      expect(loaded["node-1"]).toEqual({ x: 100, y: 200 });
      expect(loaded["node-2"]).toEqual({ x: 300, y: 400 });
    });
  });

  describe("snapshotFlowPositions", () => {
    test("copies overview positions into flow storage", () => {
      const positions = {
        "node-1": { x: 100, y: 200 },
      };
      localStorage.setItem(OVERVIEW_STORAGE_KEY, JSON.stringify(positions));

      snapshotFlowPositions("flow-1");

      const stored = localStorage.getItem(getFlowStorageKey("flow-1"));
      expect(stored).toEqual(JSON.stringify(positions));
    });
  });
});
