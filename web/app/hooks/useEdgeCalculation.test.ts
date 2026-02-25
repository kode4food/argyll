import { renderHook } from "@testing-library/react";
import { useEdgeCalculation } from "./useEdgeCalculation";
import type { Step } from "@/app/api";
import { AttributeRole, AttributeType } from "@/app/api";

describe("useEdgeCalculation", () => {
  const createStep = (
    id: string,
    requiredInputs: string[],
    optionalInputs: string[],
    outputs: string[]
  ): Step => {
    const attributes: Record<string, any> = {};
    requiredInputs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Required,
        type: AttributeType.String,
      };
    });
    optionalInputs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Optional,
        type: AttributeType.String,
      };
    });
    outputs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Output,
        type: AttributeType.String,
      };
    });

    return {
      id,
      name: `Step ${id}`,
      type: "sync",
      attributes,
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };
  };

  test("returns empty edges for empty steps", () => {
    const { result } = renderHook(() => useEdgeCalculation([]));

    expect(result.current).toEqual([]);
  });

  test("creates edge for required input dependency", () => {
    const step1 = createStep("step1", [], [], ["output1"]);
    const step2 = createStep("step2", ["output1"], [], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current).toHaveLength(1);
    expect(result.current[0]).toMatchObject({
      id: "step1-step2-output1",
      source: "step1",
      target: "step2",
      sourceHandle: "output-output1",
      targetHandle: "input-required-output1",
      type: "smoothstep",
    });
  });

  test("creates edge for optional input dependency", () => {
    const step1 = createStep("step1", [], [], ["output1"]);
    const step2 = createStep("step2", [], ["output1"], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current).toHaveLength(1);
    expect(result.current[0]).toMatchObject({
      id: "step1-step2-output1",
      source: "step1",
      target: "step2",
      sourceHandle: "output-output1",
      targetHandle: "input-optional-output1",
    });
  });

  test("creates multiple edges for multiple dependencies", () => {
    const step1 = createStep("step1", [], [], ["out1", "out2"]);
    const step2 = createStep("step2", ["out1"], ["out2"], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current).toHaveLength(2);
    expect(result.current[0].id).toBe("step1-step2-out1");
    expect(result.current[1].id).toBe("step1-step2-out2");
  });

  test("handles multiple steps with complex dependencies", () => {
    const step1 = createStep("step1", [], [], ["a"]);
    const step2 = createStep("step2", ["a"], [], ["b"]);
    const step3 = createStep("step3", ["b"], [], []);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2, step3])
    );

    expect(result.current).toHaveLength(2);
    expect(
      result.current.some((e) => e.source === "step1" && e.target === "step2")
    ).toBe(true);
    expect(
      result.current.some((e) => e.source === "step2" && e.target === "step3")
    ).toBe(true);
  });

  test("does not create edge to self", () => {
    const step1 = createStep("step1", ["output1"], [], ["output1"]);

    const { result } = renderHook(() => useEdgeCalculation([step1]));

    expect(result.current).toEqual([]);
  });

  test("applies dashed style to optional dependencies", () => {
    const step1 = createStep("step1", [], [], ["out1"]);
    const step2 = createStep("step2", [], ["out1"], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current[0].style?.strokeDasharray).toBeDefined();
  });

  test("does not apply dashed style to required dependencies", () => {
    const step1 = createStep("step1", [], [], ["out1"]);
    const step2 = createStep("step2", ["out1"], [], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current[0].style?.strokeDasharray).toBeUndefined();
  });

  test("grays out edges not in preview plan", () => {
    const step1 = createStep("step1", [], [], ["out1"]);
    const step2 = createStep("step2", ["out1"], [], []);
    const step3 = createStep("step3", [], [], ["out2"]);
    const step4 = createStep("step4", ["out2"], [], []);

    const previewStepIds = new Set(["step1", "step2"]);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2, step3, step4], previewStepIds)
    );

    const inPlanEdge = result.current.find((e) => e.source === "step1");
    const outOfPlanEdge = result.current.find((e) => e.source === "step3");

    expect(inPlanEdge?.style?.stroke).not.toContain("grayed");
    expect(outOfPlanEdge?.style?.stroke).toContain("grayed");
  });

  test("sets higher zIndex for edges in preview plan", () => {
    const step1 = createStep("step1", [], [], ["out1"]);
    const step2 = createStep("step2", ["out1"], [], []);

    const previewStepIds = new Set(["step1", "step2"]);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2], previewStepIds)
    );

    expect(result.current[0].zIndex).toBe(1001);
  });

  test("sets default zIndex for edges not in preview plan", () => {
    const step1 = createStep("step1", [], [], ["out1"]);
    const step2 = createStep("step2", ["out1"], [], []);

    const previewStepIds = new Set(["step1"]);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2], previewStepIds)
    );

    expect(result.current[0].zIndex).toBe(2);
  });

  test("keeps required edges above optional edges", () => {
    const producer = createStep("step1", [], [], ["shared"]);
    const requiredConsumer = createStep("step2", ["shared"], [], []);
    const optionalConsumer = createStep("step3", [], ["shared"], []);

    const { result } = renderHook(() =>
      useEdgeCalculation([producer, requiredConsumer, optionalConsumer])
    );

    const reqEdge = result.current.find((e) => e.target === "step2");
    const optEdge = result.current.find((e) => e.target === "step3");

    expect(reqEdge?.zIndex).toBeGreaterThan(optEdge?.zIndex ?? 0);
  });

  test("handles steps with no dependencies", () => {
    const step1 = createStep("step1", [], [], []);
    const step2 = createStep("step2", [], [], []);

    const { result } = renderHook(() => useEdgeCalculation([step1, step2]));

    expect(result.current).toEqual([]);
  });

  test("handles multiple outputs to same input", () => {
    const step1 = createStep("step1", [], [], ["shared"]);
    const step2 = createStep("step2", [], [], ["shared"]);
    const step3 = createStep("step3", ["shared"], [], []);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2, step3])
    );

    expect(result.current).toHaveLength(2);
    expect(
      result.current.some((e) => e.source === "step1" && e.target === "step3")
    ).toBe(true);
    expect(
      result.current.some((e) => e.source === "step2" && e.target === "step3")
    ).toBe(true);
  });

  test("highlights focused attribute edges while keeping others visible", () => {
    const step1 = createStep("step1", [], [], ["shared", "other"]);
    const step2 = createStep("step2", ["shared", "other"], [], []);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2], null, "shared")
    );

    const focusedEdge = result.current.find(
      (e) => e.id === "step1-step2-shared"
    );
    const dimmedEdge = result.current.find((e) => e.id === "step1-step2-other");

    expect(focusedEdge?.style?.stroke).toBe("var(--color-edge-required)");
    expect(focusedEdge?.style?.strokeWidth).toBe(2);
    expect(focusedEdge?.style?.strokeDasharray).toBeUndefined();
    expect(focusedEdge?.className).toBe("edge-focused-animated");
    expect(dimmedEdge?.style?.stroke).toBe("var(--color-edge-required)");
  });

  test("keeps normal edge styling when focused attribute has no matching edges", () => {
    const step1 = createStep("step1", [], [], ["shared"]);
    const step2 = createStep("step2", ["shared"], [], []);

    const { result } = renderHook(() =>
      useEdgeCalculation([step1, step2], null, "not_present")
    );

    expect(result.current).toHaveLength(1);
    expect(result.current[0].style?.stroke).toBe("var(--color-edge-required)");
  });
});
