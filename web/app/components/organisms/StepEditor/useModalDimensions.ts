import React, { useState, useEffect, useCallback } from "react";

export interface ModalDimensions {
  width: number;
  height: number;
}

export function useModalDimensions(
  containerRef?: React.RefObject<HTMLDivElement | null>
): { dimensions: ModalDimensions; mounted: boolean } {
  const getDimensions = useCallback((): ModalDimensions => {
    if (containerRef?.current) {
      const rect = containerRef.current.getBoundingClientRect();
      return {
        width: Math.min(rect.width * 0.8, 1000),
        height: Math.min(rect.height * 0.9, window.innerHeight * 0.9),
      };
    }
    return {
      width: Math.min(window.innerWidth * 0.9, 1000),
      height: Math.min(window.innerHeight * 0.9, 800),
    };
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
