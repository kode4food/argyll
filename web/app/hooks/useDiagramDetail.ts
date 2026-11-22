import React from "react";
import { Edge, useViewport } from "@xyflow/react";
import { useFullDetailMinZoom } from "@/app/hooks/useFullDetailMinZoom";
import {
  diagramDetailForZoom,
  edgesForDiagramDetail,
} from "@/utils/diagramDetail";

export const useDiagramDetail = (edges: Edge[]) => {
  const { zoom } = useViewport();
  const fullDetailMinZoom = useFullDetailMinZoom();
  const diagramDetail = diagramDetailForZoom(zoom, fullDetailMinZoom);
  const visibleEdges = React.useMemo(
    () => edgesForDiagramDetail(edges, diagramDetail, zoom),
    [edges, diagramDetail, zoom]
  );
  return { diagramDetail, visibleEdges };
};
