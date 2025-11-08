import { renderHook } from "@testing-library/react";
import { useEscapeKey } from "./useEscapeKey";

describe("useEscapeKey", () => {
  test("calls callback when Escape key pressed and active", () => {
    const mockCallback = jest.fn();
    renderHook(() => useEscapeKey(true, mockCallback));

    const event = new KeyboardEvent("keydown", { key: "Escape" });
    document.dispatchEvent(event);

    expect(mockCallback).toHaveBeenCalledTimes(1);
  });

  test("does not call callback when not active", () => {
    const mockCallback = jest.fn();
    renderHook(() => useEscapeKey(false, mockCallback));

    const event = new KeyboardEvent("keydown", { key: "Escape" });
    document.dispatchEvent(event);

    expect(mockCallback).not.toHaveBeenCalled();
  });

  test("does not call callback for other keys", () => {
    const mockCallback = jest.fn();
    renderHook(() => useEscapeKey(true, mockCallback));

    const event = new KeyboardEvent("keydown", { key: "Enter" });
    document.dispatchEvent(event);

    expect(mockCallback).not.toHaveBeenCalled();
  });

  test("removes event listener on unmount", () => {
    const mockCallback = jest.fn();
    const { unmount } = renderHook(() => useEscapeKey(true, mockCallback));

    unmount();

    const event = new KeyboardEvent("keydown", { key: "Escape" });
    document.dispatchEvent(event);

    expect(mockCallback).not.toHaveBeenCalled();
  });

  test("removes listener when isActive becomes false", () => {
    const mockCallback = jest.fn();
    const { rerender } = renderHook(
      ({ isActive }) => useEscapeKey(isActive, mockCallback),
      { initialProps: { isActive: true } }
    );

    rerender({ isActive: false });

    const event = new KeyboardEvent("keydown", { key: "Escape" });
    document.dispatchEvent(event);

    expect(mockCallback).not.toHaveBeenCalled();
  });
});
