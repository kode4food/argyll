import { useRef, useState, useEffect } from "react";
import { hasScrollOverflow } from "./flowFormUtils";

export function useFlowFormScrollFade(showForm: boolean): {
  sidebarListRef: React.RefObject<HTMLDivElement | null>;
  showTopFade: boolean;
  showBottomFade: boolean;
} {
  const [showTopFade, setShowTopFade] = useState(false);
  const [showBottomFade, setShowBottomFade] = useState(false);
  const sidebarListRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showForm) return;

    const el = sidebarListRef.current;
    if (!el) return;

    const handleScroll = () => {
      const { hasOverflow: overflow, atTop, atBottom } = hasScrollOverflow(el);

      if (!overflow) {
        setShowTopFade(false);
        setShowBottomFade(false);
        return;
      }

      setShowTopFade(!atTop);
      setShowBottomFade(!atBottom);
    };

    handleScroll();

    el.addEventListener("scroll", handleScroll, { passive: true });
    window.addEventListener("resize", handleScroll);

    return () => {
      el.removeEventListener("scroll", handleScroll);
      window.removeEventListener("resize", handleScroll);
    };
  }, [showForm]);

  return {
    sidebarListRef,
    showTopFade,
    showBottomFade,
  };
}
