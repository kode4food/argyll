import { renderHook, act } from "@testing-library/react";
import { Node } from "@xyflow/react";
import { useKeyboardNavigation } from "./useKeyboardNavigation";

describe("useKeyboardNavigation", () => {
  const nodes: Node[] = [
    { id: "a", position: { x: 0, y: 100 }, data: {} },
    { id: "b", position: { x: 0, y: 200 }, data: {} },
    { id: "c", position: { x: 400, y: 150 }, data: {} },
  ];

  const handleStepClick = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("handleArrowUp calls handleStepClick when next step exists", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "b", handleStepClick)
    );

    act(() => {
      result.current.handleArrowUp();
    });

    expect(handleStepClick).toHaveBeenCalledWith("a");
  });

  test("handleArrowUp does not call handleStepClick when at top", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleArrowUp();
    });

    expect(handleStepClick).not.toHaveBeenCalled();
  });

  test("handleArrowDown calls handleStepClick when next step exists", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleArrowDown();
    });

    expect(handleStepClick).toHaveBeenCalledWith("b");
  });

  test("handleArrowDown does not call handleStepClick when at bottom", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "b", handleStepClick)
    );

    act(() => {
      result.current.handleArrowDown();
    });

    expect(handleStepClick).not.toHaveBeenCalled();
  });

  test("handleArrowLeft calls handleStepClick when previous level exists", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "c", handleStepClick)
    );

    act(() => {
      result.current.handleArrowLeft();
    });

    expect(handleStepClick).toHaveBeenCalledWith("a");
  });

  test("handleArrowLeft does not call handleStepClick when at leftmost level", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleArrowLeft();
    });

    expect(handleStepClick).not.toHaveBeenCalled();
  });

  test("handleArrowRight calls handleStepClick when next level exists", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleArrowRight();
    });

    expect(handleStepClick).toHaveBeenCalledWith("c");
  });

  test("handleArrowRight does not call handleStepClick when at rightmost level", () => {
    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "c", handleStepClick)
    );

    act(() => {
      result.current.handleArrowRight();
    });

    expect(handleStepClick).not.toHaveBeenCalled();
  });

  test("handleEnter dispatches openStepEditor event when step is selected", () => {
    const dispatchSpy = jest.spyOn(document, "dispatchEvent");

    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleEnter();
    });

    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "openStepEditor",
        detail: { stepId: "a" },
      })
    );
  });

  test("handleEnter does nothing when no step is selected", () => {
    const dispatchSpy = jest.spyOn(document, "dispatchEvent");

    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, null, handleStepClick)
    );

    act(() => {
      result.current.handleEnter();
    });

    expect(dispatchSpy).not.toHaveBeenCalled();
  });

  test("handleEscape dispatches clearSelection event", () => {
    const dispatchSpy = jest.spyOn(document, "dispatchEvent");

    const { result } = renderHook(() =>
      useKeyboardNavigation(nodes, "a", handleStepClick)
    );

    act(() => {
      result.current.handleEscape();
    });

    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "clearSelection",
      })
    );
  });

  test("memoizes callbacks correctly", () => {
    const { result, rerender } = renderHook(
      ({ nodes, activeGoalStepId, handleStepClick }) =>
        useKeyboardNavigation(nodes, activeGoalStepId, handleStepClick),
      {
        initialProps: {
          nodes,
          activeGoalStepId: "a" as string | null,
          handleStepClick,
        },
      }
    );

    const initialHandlers = { ...result.current };

    rerender({
      nodes,
      activeGoalStepId: "a",
      handleStepClick,
    });

    expect(result.current.handleArrowUp).toBe(initialHandlers.handleArrowUp);
    expect(result.current.handleEnter).toBe(initialHandlers.handleEnter);
    expect(result.current.handleEscape).toBe(initialHandlers.handleEscape);
  });
});
