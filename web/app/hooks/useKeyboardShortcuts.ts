import { useEffect, useCallback } from "react";

export interface KeyboardShortcut {
  key: string;
  description: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  handler: () => void;
}

export const useKeyboardShortcuts = (
  shortcuts: KeyboardShortcut[],
  enabled: boolean = true
) => {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;

      const activeElement = document.activeElement;
      const isInputFocused =
        activeElement?.tagName === "INPUT" ||
        activeElement?.tagName === "TEXTAREA" ||
        activeElement?.getAttribute("contenteditable") === "true";

      for (const shortcut of shortcuts) {
        const keyMatches = event.key === shortcut.key;

        const ctrlMatches =
          shortcut.ctrl === undefined ? true : event.ctrlKey === shortcut.ctrl;
        const metaMatches =
          shortcut.meta === undefined ? true : event.metaKey === shortcut.meta;
        const shiftMatches =
          shortcut.shift === undefined
            ? true
            : event.shiftKey === shortcut.shift;

        if (keyMatches && ctrlMatches && metaMatches && shiftMatches) {
          if (
            (shortcut.key === "/" || shortcut.key === "?") &&
            !isInputFocused
          ) {
            event.preventDefault();
            shortcut.handler();
            return;
          }

          if (!isInputFocused || shortcut.key === "Escape") {
            event.preventDefault();
            shortcut.handler();
            return;
          }
        }
      }
    },
    [shortcuts, enabled]
  );

  useEffect(() => {
    if (enabled) {
      document.addEventListener("keydown", handleKeyDown);
      return () => document.removeEventListener("keydown", handleKeyDown);
    }
  }, [handleKeyDown, enabled]);
};
