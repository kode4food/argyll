import { useCallback } from "react";
import { useReactFlow, getViewportForBounds } from "@xyflow/react";
import { useUI } from "@/app/contexts/UIContext";
import { STEP_LAYOUT } from "@/constants/layout";

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 2;

export const useFitView = () => {
  const { diagramContainerRef, headerRef, panelRef } = useUI();
  const { getNodes, getNodesBounds, setViewport } = useReactFlow();

  return useCallback(() => {
    const nodes = getNodes();
    if (nodes.length === 0) return;

    const container = diagramContainerRef.current;
    if (!container) return;

    const containerWidth = container.clientWidth;
    const containerHeight = container.clientHeight;
    const headerHeight = headerRef.current?.offsetHeight ?? 0;
    const panelWidth = panelRef.current?.offsetWidth ?? 0;

    const visibleWidth = containerWidth - panelWidth;
    const visibleHeight = containerHeight - headerHeight;

    if (visibleWidth <= 0 || visibleHeight <= 0) return;

    const bounds = getNodesBounds(nodes);
    const viewport = getViewportForBounds(
      bounds,
      visibleWidth,
      visibleHeight,
      MIN_ZOOM,
      MAX_ZOOM,
      STEP_LAYOUT.FIT_VIEW_PADDING
    );

    void setViewport({
      x: viewport.x + panelWidth,
      y: viewport.y + headerHeight,
      zoom: viewport.zoom,
    });
  }, [
    getNodes,
    getNodesBounds,
    setViewport,
    diagramContainerRef,
    headerRef,
    panelRef,
  ]);
};
