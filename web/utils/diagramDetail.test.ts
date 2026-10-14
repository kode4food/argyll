import { Edge } from "@xyflow/react";
import {
  diagramDetailForZoom,
  edgesForDiagramDetail,
  fullDetailMinZoom,
} from "./diagramDetail";

const edge = (id: string, source: string, target: string): Edge => ({
  id,
  source,
  target,
});

describe("diagramDetail", () => {
  test("selects detail from zoom", () => {
    const minZoom = fullDetailMinZoom(16);
    expect(diagramDetailForZoom(1, minZoom)).toBe("full");
    expect(diagramDetailForZoom(minZoom, minZoom)).toBe("full");
    expect(diagramDetailForZoom(minZoom - 0.001, minZoom)).toBe("overview");
    expect(diagramDetailForZoom(0.01, minZoom)).toBe("overview");
  });

  test("derives threshold from root font size, not device pixel ratio", () => {
    expect(fullDetailMinZoom(20)).toBeLessThan(fullDetailMinZoom(16));
  });

  test("preserves attribute edges at full detail", () => {
    const edges = [edge("a", "one", "two"), edge("b", "one", "two")];
    expect(edgesForDiagramDetail(edges, "full", 1)).toBe(edges);
  });

  test("keeps midpoint connections until deeply zoomed out", () => {
    const edges = [edge("ordinary", "one", "two")];
    expect(edgesForDiagramDetail(edges, "overview", 0.2)).toEqual([
      {
        ...edges[0],
        sourceHandle: "summary-output",
        targetHandle: "summary-input",
      },
    ]);
    expect(edgesForDiagramDetail(edges, "overview", 0.199)).toEqual([]);
  });

  test("keeps plan and focused connections at minimum zoom", () => {
    const ordinary = edge("ordinary", "one", "two");
    const planned = {
      ...edge("planned", "two", "three"),
      data: { isInPlan: true },
    };
    const focused = {
      ...edge("focused", "three", "four"),
      className: "edge-focused-animated",
    };

    expect(
      edgesForDiagramDetail([ordinary, planned, focused], "overview", 0.1).map(
        ({ id }) => id
      )
    ).toEqual(["planned", "focused"]);
  });
});
