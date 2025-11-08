import { useEffect } from "react";

export const useEscapeKey = (isActive: boolean, onEscape: () => void) => {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && isActive) {
        onEscape();
      }
    };

    if (isActive) {
      document.addEventListener("keydown", handleKeyDown);
      return () => document.removeEventListener("keydown", handleKeyDown);
    }
  }, [isActive, onEscape]);
};
