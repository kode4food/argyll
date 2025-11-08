import { Viewport } from "@xyflow/react";

const VIEWPORT_STORAGE_KEY = "spuds-viewport-state";

export interface ViewportState {
  [key: string]: Viewport; // key can be workflow ID or 'overview' for overview mode
}

export const getStoredViewports = (): ViewportState => {
  try {
    const stored = localStorage.getItem(VIEWPORT_STORAGE_KEY);
    return stored ? JSON.parse(stored) : {};
  } catch (error) {
    console.warn("Failed to load viewport state from localStorage:", error);
    return {};
  }
};

export const saveViewportState = (key: string, viewport: Viewport): void => {
  try {
    const currentState = getStoredViewports();
    const newState = {
      ...currentState,
      [key]: viewport,
    };
    localStorage.setItem(VIEWPORT_STORAGE_KEY, JSON.stringify(newState));
  } catch (error) {
    console.warn("Failed to save viewport state to localStorage:", error);
  }
};

export const getViewportForKey = (key: string): Viewport | null => {
  const viewports = getStoredViewports();
  return viewports[key] || null;
};
