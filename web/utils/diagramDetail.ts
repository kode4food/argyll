import { Edge } from "@xyflow/react";

export type DiagramDetail = "full" | "overview";

export const MIN_DIAGRAM_ZOOM = 0.1;
const CONNECTIONS_MIN_ZOOM = 0.2;

// StepFooter timestamp (0.625rem) is the smallest text on screen in full
// detail, so it's first to go to mush as zoom drops
const SMALLEST_FULL_DETAIL_TEXT_REM = 0.625;
const UNREADABLE_FLOOR_PX = 5;

export const fullDetailMinZoom = (rootFontSizePx: number): number =>
  UNREADABLE_FLOOR_PX / (rootFontSizePx * SMALLEST_FULL_DETAIL_TEXT_REM);

export const diagramDetailForZoom = (
  zoom: number,
  minZoom: number
): DiagramDetail => {
  return zoom >= minZoom ? "full" : "overview";
};

export const edgesForDiagramDetail = (
  edges: Edge[],
  detail: DiagramDetail,
  zoom: number
): Edge[] => {
  if (detail === "full") return edges;
  return edges
    .filter((edge) => {
      const shouldShowEdge =
        zoom >= CONNECTIONS_MIN_ZOOM ||
        edge.data?.isInPlan === true ||
        edge.className === "edge-focused-animated";
      return shouldShowEdge;
    })
    .map((edge) => ({
      ...edge,
      sourceHandle: "summary-output",
      targetHandle: "summary-input",
    }));
};
