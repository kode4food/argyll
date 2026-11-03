import { RefObject, useLayoutEffect } from "react";

const LIST_GAP_PX = 4;

export interface AnchoredListOptions {
  isOpen: boolean;
  anchorRef: RefObject<HTMLElement | null>;
  listRef: RefObject<HTMLElement | null>;
  align?: "start" | "end";
  shouldMatchWidth?: boolean;
}

// Pins an open list under its anchor with fixed positioning, so a scrolling
// ancestor cannot clip it. An ancestor with a transform or a backdrop filter
// moves the origin fixed boxes resolve against, so that origin is measured
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

    const target = anchor.getBoundingClientRect();
    list.style.position = "fixed";
    list.style.top = "0px";
    list.style.left = "0px";
    list.style.right = "auto";
    if (shouldMatchWidth) list.style.width = `${target.width}px`;

    const origin = list.getBoundingClientRect();
    const left = align === "end" ? target.right - origin.width : target.left;
    list.style.top = `${target.bottom + LIST_GAP_PX - origin.top}px`;
    list.style.left = `${left - origin.left}px`;
  }, [isOpen, anchorRef, listRef, align, shouldMatchWidth]);
};
