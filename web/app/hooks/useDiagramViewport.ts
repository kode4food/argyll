import { useCallback, useRef, useEffect, useState } from "react";
import { Viewport } from "@xyflow/react";
import {
  saveViewportState,
  getViewportForKey,
} from "@/utils/viewportPersistence";
import { hideTooltips } from "@/utils/tooltips";

export function useDiagramViewport(viewportKey: string) {
  const canPersistRef = useRef(true);
  const [savedViewport, setSavedViewport] = useState<Viewport | null>(null);
  const [shouldFit, setShouldFit] = useState(false);

  useEffect(() => {
    const savedViewport = getViewportForKey(viewportKey);
    setSavedViewport(savedViewport);
    canPersistRef.current = false;
    setShouldFit(!savedViewport);
  }, [viewportKey]);

  const markRestored = useCallback(() => {
    canPersistRef.current = true;
  }, []);

  const markFitApplied = useCallback(() => {
    setShouldFit(false);
    canPersistRef.current = true;
  }, []);

  const handleViewportChange = useCallback(
    (viewport: Viewport) => {
      hideTooltips();
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
    markFitApplied,
  };
}
