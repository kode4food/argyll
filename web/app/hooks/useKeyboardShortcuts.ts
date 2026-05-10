import { useEffect, useCallback } from "react";

export interface KeyboardShortcut {
  key: string;
  description: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  handler: () => void;
}

const modifiersMatch = (
  event: KeyboardEvent,
  shortcut: KeyboardShortcut
): boolean => {
  if (event.key !== shortcut.key) return false;
  if (shortcut.ctrl !== undefined && event.ctrlKey !== shortcut.ctrl)
    return false;
  if (shortcut.meta !== undefined && event.metaKey !== shortcut.meta)
    return false;
  if (shortcut.shift !== undefined && event.shiftKey !== shortcut.shift)
    return false;
  return true;
};

const handleMatchedShortcut = (
  shortcut: KeyboardShortcut,
  event: KeyboardEvent,
  isInputFocused: boolean,
  shortcutsBlocked: boolean
): boolean => {
  const isHelpKey = shortcut.key === "/" || shortcut.key === "?";
  if (shortcutsBlocked && isHelpKey) return true;
  if (isHelpKey && !isInputFocused) {
    event.preventDefault();
    shortcut.handler();
    return true;
  }
  if (!isInputFocused || shortcut.key === "Escape") {
    event.preventDefault();
    shortcut.handler();
    return true;
  }
  return false;
};

export const useKeyboardShortcuts = (
  shortcuts: KeyboardShortcut[],
  enabled: boolean = true
) => {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;
      const shortcutsBlocked =
        document.querySelector("[data-ui-overlay]") !== null;
      const activeElement = document.activeElement;
      const isInputFocused =
        activeElement?.tagName === "INPUT" ||
        activeElement?.tagName === "TEXTAREA" ||
        activeElement?.getAttribute("contenteditable") === "true";

      for (const shortcut of shortcuts) {
        if (modifiersMatch(event, shortcut)) {
          if (
            handleMatchedShortcut(
              shortcut,
              event,
              isInputFocused,
              shortcutsBlocked
            )
          ) {
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
