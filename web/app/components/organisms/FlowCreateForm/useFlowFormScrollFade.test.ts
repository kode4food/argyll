import { renderHook } from "@testing-library/react";
import { useFlowFormScrollFade } from "./useFlowFormScrollFade";

describe("useFlowFormScrollFade", () => {
  it("initializes with no fades when form is not shown", () => {
    const { result } = renderHook(() => useFlowFormScrollFade(false));

    expect(result.current.showTopFade).toBe(false);
    expect(result.current.showBottomFade).toBe(false);
  });

  it("returns ref for sidebar list", () => {
    const { result } = renderHook(() => useFlowFormScrollFade(true));

    expect(result.current.sidebarListRef).toBeDefined();
    expect(result.current.sidebarListRef.current).toBeNull();
  });

  it("initializes fades when form is shown", () => {
    const { result } = renderHook(() => useFlowFormScrollFade(true));

    expect(result.current.showTopFade).toBe(false);
    expect(result.current.showBottomFade).toBe(false);
  });

  it("returns ref object with expected shape", () => {
    const { result } = renderHook(() => useFlowFormScrollFade(true));

    expect(result.current).toHaveProperty("sidebarListRef");
    expect(result.current).toHaveProperty("showTopFade");
    expect(result.current).toHaveProperty("showBottomFade");
    expect(typeof result.current.showTopFade).toBe("boolean");
    expect(typeof result.current.showBottomFade).toBe("boolean");
  });

  it("toggles between shown and not shown states", () => {
    const { result, rerender } = renderHook(
      ({ showForm }) => useFlowFormScrollFade(showForm),
      { initialProps: { showForm: false } }
    );

    expect(result.current.showTopFade).toBe(false);
    expect(result.current.showBottomFade).toBe(false);

    rerender({ showForm: true });

    expect(result.current.showTopFade).toBe(false);
    expect(result.current.showBottomFade).toBe(false);
  });

  describe("edge cases", () => {
    it("maintains state across rerenders", () => {
      const { result, rerender } = renderHook(
        ({ showForm }) => useFlowFormScrollFade(showForm),
        { initialProps: { showForm: true } }
      );

      const refObj1 = result.current.sidebarListRef;

      rerender({ showForm: true });

      const refObj2 = result.current.sidebarListRef;

      expect(refObj1).toBe(refObj2);
    });

    it("ref persists across multiple rerenders", () => {
      const { result, rerender } = renderHook(
        ({ showForm }) => useFlowFormScrollFade(showForm),
        { initialProps: { showForm: true } }
      );

      const initialRef = result.current.sidebarListRef;

      rerender({ showForm: true });
      rerender({ showForm: true });
      rerender({ showForm: true });

      expect(result.current.sidebarListRef).toBe(initialRef);
    });

    it("initializes with correct default values", () => {
      const { result } = renderHook(() => useFlowFormScrollFade(true));

      expect(result.current.showTopFade).toBe(false);
      expect(result.current.showBottomFade).toBe(false);
      expect(result.current.sidebarListRef.current).toBeNull();
    });
  });
});
