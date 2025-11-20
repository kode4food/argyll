import { renderHook } from "@testing-library/react";
import { useAutoLayout } from "./useAutoLayout";
import { Node, Edge } from "@xyflow/react";
import { ExecutionPlan, AttributeRole, AttributeType } from "../api";
import { STEP_LAYOUT } from "@/constants/layout";

// Mock dagre
let mockGraphNode: jest.Mock;
let mockSetGraph: jest.Mock;
let mockSetNode: jest.Mock;
let mockSetEdge: jest.Mock;
let mockSetDefaultEdgeLabel: jest.Mock;
let mockLayout: jest.Mock;

jest.mock("@dagrejs/dagre", () => {
  return {
    graphlib: {
      Graph: jest.fn(),
    },
    layout: jest.fn(),
  };
});

describe("useAutoLayout", () => {
  beforeEach(() => {
    const dagre = require("@dagrejs/dagre");

    mockGraphNode = jest.fn().mockReturnValue({ x: 200, y: 150 });
    mockSetGraph = jest.fn();
    mockSetNode = jest.fn();
    mockSetEdge = jest.fn();
    mockSetDefaultEdgeLabel = jest.fn();
    mockLayout = dagre.layout as jest.Mock;

    dagre.graphlib.Graph.mockImplementation(() => ({
      setGraph: mockSetGraph,
      setNode: mockSetNode,
      setEdge: mockSetEdge,
      setDefaultEdgeLabel: mockSetDefaultEdgeLabel,
      node: mockGraphNode,
    }));

    jest.clearAllMocks();
    mockGraphNode.mockReturnValue({ x: 200, y: 150 });
  });

  const createNode = (id: string, attributes?: any): Node => ({
    id,
    position: { x: 0, y: 0 },
    data: {
      step: {
        id,
        name: `Step ${id}`,
        type: "sync",
        version: "1.0.0",
        attributes,
        http: { endpoint: "http://test", timeout: 5000 },
      },
    },
    type: "step",
  });

  test("returns original nodes when plan is null", () => {
    const nodes = [createNode("step-1")];
    const edges: Edge[] = [];

    const { result } = renderHook(() => useAutoLayout(nodes, edges, null));

    expect(result.current).toEqual(nodes);
    expect(mockSetGraph).not.toHaveBeenCalled();
  });

  test("returns original nodes when plan is undefined", () => {
    const nodes = [createNode("step-1")];
    const edges: Edge[] = [];

    const { result } = renderHook(() => useAutoLayout(nodes, edges, undefined));

    expect(result.current).toEqual(nodes);
    expect(mockSetGraph).not.toHaveBeenCalled();
  });

  test("returns original nodes when nodes array is empty", () => {
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    const { result } = renderHook(() => useAutoLayout([], [], plan));

    expect(result.current).toEqual([]);
    expect(mockSetGraph).not.toHaveBeenCalled();
  });

  test("sets up dagre graph with default config", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetDefaultEdgeLabel).toHaveBeenCalled();
    expect(mockSetGraph).toHaveBeenCalledWith({
      rankdir: "LR",
      ranksep: 50,
      nodesep: 15,
      marginx: 20,
      marginy: 20,
    });
  });

  test("applies custom layout config", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };
    const config = {
      rankdir: "TB" as const,
      rankSep: 100,
      nodeSep: 30,
    };

    renderHook(() => useAutoLayout(nodes, [], plan, config));

    expect(mockSetGraph).toHaveBeenCalledWith({
      rankdir: "TB",
      ranksep: 100,
      nodesep: 30,
      marginx: 20,
      marginy: 20,
    });
  });

  test("calculates base height for node without attributes", () => {
    const node = createNode("step-1", undefined);
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout([node], [], plan));

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 320,
      height: STEP_LAYOUT.WIDGET_BASE_HEIGHT,
    });
  });

  test("calculates height for node with required attributes", () => {
    const node = createNode("step-1", {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      input2: { role: AttributeRole.Required, type: AttributeType.String },
    });
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout([node], [], plan));

    const expectedHeight =
      STEP_LAYOUT.WIDGET_BASE_HEIGHT +
      STEP_LAYOUT.SECTION_HEIGHT +
      2 * STEP_LAYOUT.ARG_LINE_HEIGHT;

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 320,
      height: expectedHeight,
    });
  });

  test("calculates height for node with optional attributes", () => {
    const node = createNode("step-1", {
      opt1: { role: AttributeRole.Optional, type: AttributeType.String },
    });
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout([node], [], plan));

    const expectedHeight =
      STEP_LAYOUT.WIDGET_BASE_HEIGHT +
      STEP_LAYOUT.SECTION_HEIGHT +
      STEP_LAYOUT.ARG_LINE_HEIGHT;

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 320,
      height: expectedHeight,
    });
  });

  test("calculates height for node with output attributes", () => {
    const node = createNode("step-1", {
      out1: { role: AttributeRole.Output, type: AttributeType.String },
      out2: { role: AttributeRole.Output, type: AttributeType.String },
      out3: { role: AttributeRole.Output, type: AttributeType.String },
    });
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout([node], [], plan));

    const expectedHeight =
      STEP_LAYOUT.WIDGET_BASE_HEIGHT +
      STEP_LAYOUT.SECTION_HEIGHT +
      3 * STEP_LAYOUT.ARG_LINE_HEIGHT;

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 320,
      height: expectedHeight,
    });
  });

  test("calculates height for node with mixed attributes", () => {
    const node = createNode("step-1", {
      req1: { role: AttributeRole.Required, type: AttributeType.String },
      opt1: { role: AttributeRole.Optional, type: AttributeType.String },
      out1: { role: AttributeRole.Output, type: AttributeType.String },
    });
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout([node], [], plan));

    const expectedHeight =
      STEP_LAYOUT.WIDGET_BASE_HEIGHT +
      3 * STEP_LAYOUT.SECTION_HEIGHT +
      3 * STEP_LAYOUT.ARG_LINE_HEIGHT;

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 320,
      height: expectedHeight,
    });
  });

  test("creates edges from plan attribute dependencies", () => {
    const nodes = [createNode("step-1"), createNode("step-2")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1"],
          consumers: ["step-2"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).toHaveBeenCalledWith("step-1", "step-2");
  });

  test("creates multiple edges for multiple consumers", () => {
    const nodes = [
      createNode("step-1"),
      createNode("step-2"),
      createNode("step-3"),
    ];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1"],
          consumers: ["step-2", "step-3"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).toHaveBeenCalledWith("step-1", "step-2");
    expect(mockSetEdge).toHaveBeenCalledWith("step-1", "step-3");
  });

  test("creates multiple edges for multiple providers", () => {
    const nodes = [
      createNode("step-1"),
      createNode("step-2"),
      createNode("step-3"),
    ];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1", "step-2"],
          consumers: ["step-3"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).toHaveBeenCalledWith("step-1", "step-3");
    expect(mockSetEdge).toHaveBeenCalledWith("step-2", "step-3");
  });

  test("skips edges when provider node not found", () => {
    const nodes = [createNode("step-2")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1"],
          consumers: ["step-2"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).not.toHaveBeenCalled();
  });

  test("skips edges when consumer node not found", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1"],
          consumers: ["step-2"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).not.toHaveBeenCalled();
  });

  test("handles attributes without providers", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          consumers: ["step-1"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).not.toHaveBeenCalled();
  });

  test("handles attributes without consumers", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: {
          providers: ["step-1"],
        },
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).not.toHaveBeenCalled();
  });

  test("handles null attribute dependencies", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {
        attr1: null as any,
      },
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockSetEdge).not.toHaveBeenCalled();
  });

  test("calls dagre layout", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    renderHook(() => useAutoLayout(nodes, [], plan));

    expect(mockLayout).toHaveBeenCalled();
  });

  test("returns nodes with calculated positions", () => {
    mockGraphNode.mockReturnValue({ x: 400, y: 300 });

    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    const { result } = renderHook(() => useAutoLayout(nodes, [], plan));

    expect(result.current[0].position).toEqual({
      x: 400 - 320 / 2,
      y: 300 - STEP_LAYOUT.WIDGET_BASE_HEIGHT / 2,
    });
  });

  test("preserves node when dagre has no position for it", () => {
    mockGraphNode.mockReturnValue(undefined);

    const nodes = [createNode("step-1", undefined)];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };

    const { result } = renderHook(() => useAutoLayout(nodes, [], plan));

    expect(result.current[0]).toEqual(nodes[0]);
  });

  test("uses custom nodeWidth from config", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };
    const config = { nodeWidth: 400 };

    renderHook(() => useAutoLayout(nodes, [], plan, config));

    expect(mockSetNode).toHaveBeenCalledWith("step-1", {
      width: 400,
      height: STEP_LAYOUT.WIDGET_BASE_HEIGHT,
    });
  });

  test("memoizes result based on nodes, plan, and config", () => {
    const nodes = [createNode("step-1")];
    const plan: ExecutionPlan = {
      steps: {},
      attributes: {},
      goals: [],
      required: [],
    };
    const config = {};

    const { rerender } = renderHook(
      ({ n, p, c }) => useAutoLayout(n, [], p, c),
      { initialProps: { n: nodes, p: plan, c: config } }
    );

    const callCount = mockSetGraph.mock.calls.length;

    // Re-render with same props
    rerender({ n: nodes, p: plan, c: config });
    expect(mockSetGraph.mock.calls.length).toBe(callCount);

    // Re-render with different nodes
    const newNodes = [createNode("step-2")];
    rerender({ n: newNodes, p: plan, c: config });
    expect(mockSetGraph.mock.calls.length).toBeGreaterThan(callCount);
  });

  test("handles plan without attributes property", () => {
    const nodes = [createNode("step-1")];
    const plan = {
      steps: {},
      goals: [],
      required: [],
    } as ExecutionPlan;

    const { result } = renderHook(() => useAutoLayout(nodes, [], plan));

    expect(result.current).toBeDefined();
    expect(mockSetEdge).not.toHaveBeenCalled();
  });
});
