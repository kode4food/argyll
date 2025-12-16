import { useCallback, useRef, useEffect, useState } from "react";
import { Viewport } from "@xyflow/react";
import { saveViewportState, getViewportForKey } from "./viewportPersistence";

export function useDiagramViewport(viewportKey: string) {
  const initialViewportSet = useRef(false);
  const [shouldFit, setShouldFit] = useState(false);

  useEffect(() => {
    initialViewportSet.current = false;
    const savedViewport = getViewportForKey(viewportKey);
    setShouldFit(!savedViewport);
  }, [viewportKey]);

  const handleViewportChange = useCallback(
    (viewport: Viewport) => {
      const event = new CustomEvent("hideTooltips");
      document.dispatchEvent(event);
      saveViewportState(viewportKey, viewport);
      if (!initialViewportSet.current) {
        initialViewportSet.current = true;
      }
    },
    [viewportKey]
  );

  return {
    handleViewportChange,
    shouldFitView: shouldFit,
  };
}
