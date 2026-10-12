import { Edge } from "@xyflow/react";
import { diagramDetailForZoom, edgesForDiagramDetail } from "./diagramDetail";

const edge = (id: string, source: string, target: string): Edge => ({
  id,
  source,
  target,
});

describe("diagramDetail", () => {
  test("selects detail from zoom", () => {
    expect(diagramDetailForZoom(1)).toBe("full");
    expect(diagramDetailForZoom(0.6)).toBe("full");
    expect(diagramDetailForZoom(0.5)).toBe("full");
    expect(diagramDetailForZoom(0.499)).toBe("overview");
    expect(diagramDetailForZoom(0.2)).toBe("overview");
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
