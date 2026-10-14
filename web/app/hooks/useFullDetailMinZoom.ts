import { useState, useEffect } from "react";
import { fullDetailMinZoom } from "@/utils/diagramDetail";

// Recomputes on resize since the root font size can change (browser zoom)
export const useFullDetailMinZoom = (): number => {
  const [minZoom, setMinZoom] = useState(() => compute());

  useEffect(() => {
    const recompute = () => setMinZoom(compute());
    window.addEventListener("resize", recompute);
    return () => window.removeEventListener("resize", recompute);
  }, []);

  return minZoom;
};

const DEFAULT_ROOT_FONT_SIZE_PX = 16;

function compute(): number {
  const parsed = parseFloat(
    getComputedStyle(document.documentElement).fontSize
  );
  const rootFontSizePx = Number.isNaN(parsed)
    ? DEFAULT_ROOT_FONT_SIZE_PX
    : parsed;
  return fullDetailMinZoom(rootFontSizePx);
}
