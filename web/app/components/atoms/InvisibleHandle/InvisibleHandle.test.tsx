import { render } from "@testing-library/react";
import { Position } from "@xyflow/react";
import InvisibleHandle from "./InvisibleHandle";

// Mock @xyflow/react
jest.mock("@xyflow/react", () => ({
  Handle: ({ id, type, position, style, className }: any) => (
    <div
      data-testid={`handle-${id}`}
      data-type={type}
      data-position={position}
      style={style}
      className={className}
    />
  ),
  Position: {
    Left: "left",
    Right: "right",
    Top: "top",
    Bottom: "bottom",
  },
}));

describe("InvisibleHandle", () => {
  test("renders handle with correct id", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="source"
        position={Position.Left}
        top={100}
        argName="testArg"
      />
    );
    expect(getByTestId("handle-test-handle")).toBeInTheDocument();
  });

  test("applies left position class", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="source"
        position={Position.Left}
        top={100}
        argName="testArg"
      />
    );
    const handle = getByTestId("handle-test-handle");
    expect(handle.className).toContain("invisible-handle-left");
  });

  test("applies right position class", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="target"
        position={Position.Right}
        top={100}
        argName="testArg"
      />
    );
    const handle = getByTestId("handle-test-handle");
    expect(handle.className).toContain("invisible-handle-right");
  });

  test("sets top style correctly", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="source"
        position={Position.Left}
        top={250}
        argName="testArg"
      />
    );
    const handle = getByTestId("handle-test-handle");
    expect(handle.style.top).toBe("250px");
  });

  test("renders source type handle", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="source"
        position={Position.Left}
        top={100}
        argName="testArg"
      />
    );
    const handle = getByTestId("handle-test-handle");
    expect(handle.getAttribute("data-type")).toBe("source");
  });

  test("renders target type handle", () => {
    const { getByTestId } = render(
      <InvisibleHandle
        id="test-handle"
        type="target"
        position={Position.Right}
        top={100}
        argName="testArg"
      />
    );
    const handle = getByTestId("handle-test-handle");
    expect(handle.getAttribute("data-type")).toBe("target");
  });
});
