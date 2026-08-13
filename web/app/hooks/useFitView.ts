import { useCallback } from "react";
import { useReactFlow, getViewportForBounds } from "@xyflow/react";
import { useUI } from "@/app/contexts/UIContext";
import { stepLayout } from "@/constants/layout";

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 2;

export const useFitView = () => {
  const { diagramContainerRef, panelRef } = useUI();
  const { getNodes, getNodesBounds, setViewport } = useReactFlow();

  return useCallback(() => {
    const nodes = getNodes();
    if (nodes.length === 0) return;

    const container = diagramContainerRef.current;
    if (!container) return;

    const panelWidth = panelRef.current?.offsetWidth ?? 0;

    const visibleWidth = container.clientWidth - panelWidth;

    if (visibleWidth <= 0 || container.clientHeight <= 0) return;

    const bounds = getNodesBounds(nodes);
    const viewport = getViewportForBounds(
      bounds,
      visibleWidth,
      container.clientHeight,
      MIN_ZOOM,
      MAX_ZOOM,
      stepLayout.FIT_VIEW_PADDING
    );

    void setViewport({
      x: viewport.x + panelWidth,
      y: viewport.y,
      zoom: viewport.zoom,
    });
  }, [getNodes, getNodesBounds, setViewport, diagramContainerRef, panelRef]);
};
