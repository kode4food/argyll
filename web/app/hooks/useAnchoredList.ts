import { RefObject, useLayoutEffect } from "react";

const LIST_GAP_PX = 4;

export interface AnchoredListOptions {
  isOpen: boolean;
  anchorRef: RefObject<HTMLElement | null>;
  listRef: RefObject<HTMLElement | null>;
  align?: "start" | "end";
  shouldMatchWidth?: boolean;
}

// Positions an open list next to its anchor and fits it within the viewport.
// Measuring the origin accounts for transforms on the list's ancestors.
export const useAnchoredList = ({
  isOpen,
  anchorRef,
  listRef,
  align = "start",
  shouldMatchWidth = false,
}: AnchoredListOptions): void => {
  useLayoutEffect(() => {
    const anchor = anchorRef.current;
    const list = listRef.current;
    if (!isOpen || !anchor || !list) return;

    list.style.position = "fixed";
    list.style.right = "auto";
    list.style.boxSizing = "border-box";
    list.style.overflowY = "auto";

    const position = () => {
      const target = anchor.getBoundingClientRect();
      list.style.top = "0px";
      list.style.left = "0px";
      list.style.maxHeight = "none";
      if (shouldMatchWidth) list.style.width = `${target.width}px`;

      const origin = list.getBoundingClientRect();
      const below = window.innerHeight - target.bottom - LIST_GAP_PX;
      const above = target.top - LIST_GAP_PX;
      const opensUp = origin.height > below && above > below;
      const maxHeight = Math.max(0, opensUp ? above : below);
      list.style.maxHeight = `${maxHeight}px`;

      const top = opensUp
        ? target.top - LIST_GAP_PX - Math.min(origin.height, maxHeight)
        : target.bottom + LIST_GAP_PX;
      const left = align === "end" ? target.right - origin.width : target.left;
      list.style.top = `${top - origin.top}px`;
      list.style.left = `${left - origin.left}px`;
    };
    const onScroll = (event: Event) => {
      if (event.target === list) return;
      position();
    };
    position();
    window.addEventListener("resize", position);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      window.removeEventListener("resize", position);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [isOpen, anchorRef, listRef, align, shouldMatchWidth]);
};
