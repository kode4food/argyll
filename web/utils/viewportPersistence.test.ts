import {
  getStoredViewports,
  saveViewportState,
  getViewportForKey,
} from "./viewportPersistence";
import { Viewport } from "@xyflow/react";

describe("viewportPersistence", () => {
  const STORAGE_KEY = "spuds-viewport-state";

  beforeEach(() => {
    localStorage.clear();
    jest.spyOn(console, "warn").mockImplementation();
  });

  afterEach(() => {
    localStorage.clear();
    jest.restoreAllMocks();
  });

  describe("getStoredViewports", () => {
    test("returns empty object when no data stored", () => {
      const result = getStoredViewports();
      expect(result).toEqual({});
    });

    test("loads viewport state from localStorage", () => {
      const viewports = {
        "flow-1": { x: 100, y: 200, zoom: 1.5 },
        overview: { x: 0, y: 0, zoom: 1 },
      };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(viewports));

      const result = getStoredViewports();

      expect(result).toEqual(viewports);
    });

    test("returns empty object on parse error", () => {
      localStorage.setItem(STORAGE_KEY, "invalid json");

      const result = getStoredViewports();

      expect(result).toEqual({});
      expect(console.warn).toHaveBeenCalledWith(
        "Failed to load viewport state from localStorage:",
        expect.any(Error)
      );
    });

    test("handles empty stored object", () => {
      localStorage.setItem(STORAGE_KEY, "{}");

      const result = getStoredViewports();

      expect(result).toEqual({});
    });
  });

  describe("saveViewportState", () => {
    test("saves viewport state to localStorage", () => {
      const viewport: Viewport = { x: 100, y: 200, zoom: 1.5 };

      saveViewportState("flow-1", viewport);

      const stored = localStorage.getItem(STORAGE_KEY);
      expect(stored).toBeTruthy();
      const parsed = JSON.parse(stored!);
      expect(parsed["flow-1"]).toEqual(viewport);
    });

    test("preserves existing viewports when saving new one", () => {
      const viewport1: Viewport = { x: 100, y: 200, zoom: 1.5 };
      const viewport2: Viewport = { x: 300, y: 400, zoom: 2.0 };

      saveViewportState("flow-1", viewport1);
      saveViewportState("flow-2", viewport2);

      const stored = localStorage.getItem(STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["flow-1"]).toEqual(viewport1);
      expect(parsed["flow-2"]).toEqual(viewport2);
    });

    test("overwrites existing viewport for same key", () => {
      const viewport1: Viewport = { x: 100, y: 200, zoom: 1.5 };
      const viewport2: Viewport = { x: 300, y: 400, zoom: 2.0 };

      saveViewportState("flow-1", viewport1);
      saveViewportState("flow-1", viewport2);

      const stored = localStorage.getItem(STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["flow-1"]).toEqual(viewport2);
    });

    test("handles overview key", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 };

      saveViewportState("overview", viewport);

      const stored = localStorage.getItem(STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["overview"]).toEqual(viewport);
    });

    test("handles fractional viewport values", () => {
      const viewport: Viewport = { x: 123.456, y: 789.012, zoom: 1.23 };

      saveViewportState("flow-1", viewport);

      const stored = localStorage.getItem(STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["flow-1"]).toEqual(viewport);
    });

    test("handles negative viewport values", () => {
      const viewport: Viewport = { x: -100, y: -200, zoom: 0.5 };

      saveViewportState("flow-1", viewport);

      const stored = localStorage.getItem(STORAGE_KEY);
      const parsed = JSON.parse(stored!);
      expect(parsed["flow-1"]).toEqual(viewport);
    });

    test("logs warning on storage error", () => {
      const viewport: Viewport = { x: 100, y: 200, zoom: 1.5 };
      jest.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new Error("Storage full");
      });

      saveViewportState("flow-1", viewport);

      expect(console.warn).toHaveBeenCalledWith(
        "Failed to save viewport state to localStorage:",
        expect.any(Error)
      );
    });
  });

  describe("getViewportForKey", () => {
    test("returns viewport for existing key", () => {
      const viewport: Viewport = { x: 100, y: 200, zoom: 1.5 };
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ "flow-1": viewport })
      );

      const result = getViewportForKey("flow-1");

      expect(result).toEqual(viewport);
    });

    test("returns null for non-existent key", () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ "flow-1": {} }));

      const result = getViewportForKey("flow-2");

      expect(result).toBeNull();
    });

    test("returns null when no data stored", () => {
      const result = getViewportForKey("flow-1");

      expect(result).toBeNull();
    });

    test("returns viewport for overview key", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 };
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ overview: viewport }));

      const result = getViewportForKey("overview");

      expect(result).toEqual(viewport);
    });
  });

  describe("integration", () => {
    test("saves and retrieves viewport correctly", () => {
      const viewport: Viewport = { x: 100, y: 200, zoom: 1.5 };

      saveViewportState("flow-1", viewport);
      const retrieved = getViewportForKey("flow-1");

      expect(retrieved).toEqual(viewport);
    });

    test("manages multiple viewports independently", () => {
      const viewport1: Viewport = { x: 100, y: 200, zoom: 1.5 };
      const viewport2: Viewport = { x: 300, y: 400, zoom: 2.0 };
      const overview: Viewport = { x: 0, y: 0, zoom: 1 };

      saveViewportState("flow-1", viewport1);
      saveViewportState("flow-2", viewport2);
      saveViewportState("overview", overview);

      expect(getViewportForKey("flow-1")).toEqual(viewport1);
      expect(getViewportForKey("flow-2")).toEqual(viewport2);
      expect(getViewportForKey("overview")).toEqual(overview);
    });

    test("updates existing viewport without affecting others", () => {
      const viewport1: Viewport = { x: 100, y: 200, zoom: 1.5 };
      const viewport2: Viewport = { x: 300, y: 400, zoom: 2.0 };
      const viewport1Updated: Viewport = { x: 500, y: 600, zoom: 3.0 };

      saveViewportState("flow-1", viewport1);
      saveViewportState("flow-2", viewport2);
      saveViewportState("flow-1", viewport1Updated);

      expect(getViewportForKey("flow-1")).toEqual(viewport1Updated);
      expect(getViewportForKey("flow-2")).toEqual(viewport2);
    });
  });
});
