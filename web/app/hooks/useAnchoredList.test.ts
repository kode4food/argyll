import { fireEvent, renderHook } from "@testing-library/react";
import { useAnchoredList } from "./useAnchoredList";

const withRect = (rect: Partial<DOMRect>): HTMLElement => {
  const el = document.createElement("div");
  el.getBoundingClientRect = () =>
    ({ top: 0, left: 0, right: 0, bottom: 0, width: 0, ...rect }) as DOMRect;
  return el;
};

const refs = () => ({
  anchorRef: {
    current: withRect({ left: 100, right: 200, bottom: 50, width: 100 }),
  },
  // A transformed or filtered ancestor offsets where fixed boxes land
  listRef: { current: withRect({ top: 20, left: 30, width: 80 }) },
});

describe("useAnchoredList", () => {
  test("pins the list under its anchor, correcting for the origin", () => {
    const { anchorRef, listRef } = refs();
    renderHook(() =>
      useAnchoredList({
        isOpen: true,
        anchorRef,
        listRef,
        shouldMatchWidth: true,
      })
    );

    expect(listRef.current.style.position).toBe("fixed");
    expect(listRef.current.style.top).toBe("34px");
    expect(listRef.current.style.left).toBe("70px");
    expect(listRef.current.style.width).toBe("100px");
  });

  test("aligns the list's end with the anchor's", () => {
    const { anchorRef, listRef } = refs();
    renderHook(() =>
      useAnchoredList({ isOpen: true, anchorRef, listRef, align: "end" })
    );

    expect(listRef.current.style.left).toBe("90px");
    expect(listRef.current.style.width).toBe("");
  });

  test("leaves a closed list alone", () => {
    const { anchorRef, listRef } = refs();
    renderHook(() => useAnchoredList({ isOpen: false, anchorRef, listRef }));

    expect(listRef.current.style.position).toBe("");
  });

  test("opens above the anchor when the viewport ends below it", () => {
    const anchorRef = {
      current: withRect({
        top: window.innerHeight - 40,
        bottom: window.innerHeight - 8,
      }),
    };
    const listRef = { current: withRect({ height: window.innerHeight }) };
    renderHook(() => useAnchoredList({ isOpen: true, anchorRef, listRef }));

    expect(listRef.current.style.top).toBe("0px");
    expect(listRef.current.style.maxHeight).toBe(
      `${window.innerHeight - 44}px`
    );
    expect(listRef.current.style.overflowY).toBe("auto");
  });

  test("follows the anchor when a container scrolls", () => {
    const { anchorRef, listRef } = refs();
    renderHook(() => useAnchoredList({ isOpen: true, anchorRef, listRef }));
    anchorRef.current.getBoundingClientRect = withRect({
      top: 75,
      bottom: 100,
    }).getBoundingClientRect;
    fireEvent.scroll(window);

    expect(listRef.current.style.top).toBe("84px");
  });
});
