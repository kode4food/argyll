import { Edge } from "@xyflow/react";

export type DiagramDetail = "full" | "overview";

export const MIN_DIAGRAM_ZOOM = 0.1;
const FULL_DETAIL_MIN_ZOOM = 0.5;
const CONNECTIONS_MIN_ZOOM = 0.2;

export const diagramDetailForZoom = (zoom: number): DiagramDetail => {
  return zoom >= FULL_DETAIL_MIN_ZOOM ? "full" : "overview";
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
