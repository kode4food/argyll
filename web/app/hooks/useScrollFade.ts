import { useRef, useState, useEffect } from "react";

export function useScrollFade(active: boolean = true): {
  scrollRef: React.RefObject<HTMLDivElement | null>;
  showTopFade: boolean;
  showBottomFade: boolean;
} {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [showTopFade, setShowTopFade] = useState(false);
  const [showBottomFade, setShowBottomFade] = useState(false);

  useEffect(() => {
    if (!active) return;
    const el = scrollRef.current;
    if (!el) return;

    const update = () => {
      const { scrollTop, scrollHeight, clientHeight } = el;
      const overflow = scrollHeight > clientHeight;
      setShowTopFade(overflow && scrollTop > 1);
      setShowBottomFade(
        overflow && scrollTop < scrollHeight - clientHeight - 1
      );
    };

    update();
    const timer = setTimeout(update, 0);
    el.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update);

    return () => {
      clearTimeout(timer);
      el.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [active]);

  return { scrollRef, showTopFade, showBottomFade };
}
