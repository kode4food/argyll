import { renderHook, act } from "@testing-library/react";
import { useThrottledValue } from "./useThrottledValue";

jest.useFakeTimers();

describe("useThrottledValue", () => {
  afterEach(() => {
    jest.clearAllTimers();
  });

  test("returns initial value immediately", () => {
    const { result } = renderHook(() => useThrottledValue("initial", 1000));
    expect(result.current).toBe("initial");
  });

  test("throttles value updates", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useThrottledValue(value, delay),
      { initialProps: { value: "initial", delay: 1000 } }
    );

    expect(result.current).toBe("initial");

    rerender({ value: "updated", delay: 1000 });
    expect(result.current).toBe("initial");

    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current).toBe("updated");
  });

  test("clears previous timeout on rapid updates", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useThrottledValue(value, delay),
      { initialProps: { value: "initial", delay: 1000 } }
    );

    rerender({ value: "update1", delay: 1000 });
    act(() => {
      jest.advanceTimersByTime(500);
    });

    rerender({ value: "update2", delay: 1000 });
    act(() => {
      jest.advanceTimersByTime(500);
    });

    expect(result.current).toBe("initial");

    act(() => {
      jest.advanceTimersByTime(500);
    });

    expect(result.current).toBe("update2");
  });

  test("handles different delay values", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useThrottledValue(value, delay),
      { initialProps: { value: "initial", delay: 500 } }
    );

    rerender({ value: "updated", delay: 500 });
    act(() => {
      jest.advanceTimersByTime(500);
    });

    expect(result.current).toBe("updated");
  });

  test("cleans up timeout on unmount", () => {
    const clearTimeoutSpy = jest.spyOn(global, "clearTimeout");
    const { unmount } = renderHook(() => useThrottledValue("value", 1000));

    unmount();

    expect(clearTimeoutSpy).toHaveBeenCalled();
    clearTimeoutSpy.mockRestore();
  });

  test("works with different value types", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useThrottledValue(value, delay),
      { initialProps: { value: 42, delay: 1000 } }
    );

    expect(result.current).toBe(42);

    rerender({ value: 100, delay: 1000 });
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    expect(result.current).toBe(100);
  });
});
