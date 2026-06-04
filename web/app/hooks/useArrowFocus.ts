import React from "react";

const FOCUS_ITEM_SELECTOR =
  '[data-arrow-focus-item="true"]:not([aria-disabled="true"])';

const useArrowFocus = () => {
  const handleArrowFocus = React.useCallback(
    (e: React.KeyboardEvent<HTMLElement>) => {
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;

      const scope = e.currentTarget;
      const active = e.target;
      const items = Array.from(
        scope.querySelectorAll<HTMLElement>(FOCUS_ITEM_SELECTOR)
      );
      const currentIdx = items.findIndex((item) => item === active);
      if (currentIdx < 0) return;

      const nextIdx =
        e.key === "ArrowDown"
          ? Math.min(currentIdx + 1, items.length - 1)
          : Math.max(currentIdx - 1, 0);
      if (nextIdx === currentIdx) return;
      e.preventDefault();
      items[nextIdx]?.focus();
    },
    []
  );

  return handleArrowFocus;
};

export default useArrowFocus;
