import { useCallback, useRef, useEffect, useState } from "react";
import { Viewport } from "@xyflow/react";
import {
  saveViewportState,
  getViewportForKey,
} from "@/utils/viewportPersistence";

export function useDiagramViewport(viewportKey: string) {
  const canPersistRef = useRef(true);
  const [savedViewport, setSavedViewport] = useState<Viewport | null>(null);
  const [shouldFit, setShouldFit] = useState(false);

  useEffect(() => {
    const savedViewport = getViewportForKey(viewportKey);
    setSavedViewport(savedViewport);
    canPersistRef.current = !savedViewport;
    setShouldFit(!savedViewport);
  }, [viewportKey]);

  const markRestored = useCallback(() => {
    canPersistRef.current = true;
  }, []);

  const handleViewportChange = useCallback(
    (viewport: Viewport) => {
      const event = new CustomEvent("hideTooltips");
      document.dispatchEvent(event);
      if (!canPersistRef.current) {
        return;
      }
      saveViewportState(viewportKey, viewport);
    },
    [viewportKey]
  );

  return {
    handleViewportChange,
    shouldFitView: shouldFit,
    savedViewport,
    markRestored,
  };
}
