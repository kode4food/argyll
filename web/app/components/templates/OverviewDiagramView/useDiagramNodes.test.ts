import { renderHook } from "@testing-library/react";
import { Node } from "@xyflow/react";
import { useDiagramNodes } from "./useDiagramNodes";
import { loadNodePositions, saveNodePositions } from "@/utils/nodePositioning";

jest.mock("@xyflow/react", () => ({
  useNodesState: (nodes: Node[]) => [nodes, jest.fn(), jest.fn()],
}));

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(),
  saveNodePositions: jest.fn(),
}));

describe("useDiagramNodes", () => {
  const scope = { type: "overview" as const };
  const node = (id: string): Node => ({
    id,
    position: { x: 1, y: 2 },
    data: {},
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("saves positions of Steps placed without one", () => {
    (loadNodePositions as jest.Mock).mockReturnValue({ a: { x: 0, y: 0 } });

    renderHook(() => useDiagramNodes([node("a"), node("b")], scope));

    expect(saveNodePositions).toHaveBeenCalledWith([node("b")], scope);
  });

  test("leaves saved positions alone", () => {
    (loadNodePositions as jest.Mock).mockReturnValue({ a: { x: 0, y: 0 } });

    renderHook(() => useDiagramNodes([node("a")], scope));

    expect(saveNodePositions).not.toHaveBeenCalled();
  });
});
