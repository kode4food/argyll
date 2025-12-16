import { useState, useEffect, useCallback } from "react";

export interface ModalDimensions {
  width: number;
  minHeight: number;
}

export function useModalDimensions(
  containerRef?: React.RefObject<HTMLDivElement | null>
): { dimensions: ModalDimensions; mounted: boolean } {
  const getDimensions = useCallback((): ModalDimensions => {
    if (containerRef?.current) {
      const rect = containerRef.current.getBoundingClientRect();
      return {
        width: Math.min(rect.width * 0.8, 1000),
        minHeight: rect.height * 0.9,
      };
    }
    return { width: 1000, minHeight: 800 };
  }, [containerRef]);

  const [dimensions, setDimensions] = useState<ModalDimensions>(getDimensions);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const dims = getDimensions();
    setDimensions(dims);
  }, [getDimensions]);

  return { dimensions, mounted };
}
