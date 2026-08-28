import { act, renderHook } from "@testing-library/react";
import { useScrollFade } from "@/app/hooks/useScrollFade";

function attachScrollable(el: HTMLDivElement | null) {
  if (!el) return;
  Object.defineProperty(el, "scrollHeight", { value: 500, configurable: true });
  Object.defineProperty(el, "clientHeight", { value: 100, configurable: true });
}

describe("useScrollFade", () => {
  it("clears fades when it goes inactive", () => {
    const { result, rerender } = renderHook(
      ({ active }) => useScrollFade(active),
      { initialProps: { active: false } }
    );

    act(() => {
      const el = document.createElement("div");
      attachScrollable(el);
      result.current.scrollRef.current = el;
    });
    rerender({ active: true });
    expect(result.current.showBottomFade).toBe(true);

    rerender({ active: false });
    expect(result.current.showBottomFade).toBe(false);
    expect(result.current.showTopFade).toBe(false);
  });
});
