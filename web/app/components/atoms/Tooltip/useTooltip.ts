import React, { useEffect, useLayoutEffect, useRef, useState } from "react";
import { UI } from "@/constants/common";
import { HIDE_TOOLTIPS_EVENT } from "@/utils/tooltips";

export const useTooltip = (children: React.ReactNode) => {
  const [isVisible, setIsVisible] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ top: 0, left: 0 });
  const [isBelow, setIsBelow] = useState(false);
  const [portalRoot, setPortalRoot] = useState<Element>(document.body);
  const containerRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const handleMouseEnter = () => {
    const container = containerRef.current;
    if (!container) return;
    // A modal dialog sits in the top layer, above anything portalled to body
    setPortalRoot(container.closest("dialog") ?? document.body);
    setTooltipPosition({
      top: 0,
      left: container.getBoundingClientRect().left,
    });
    setIsVisible(true);
  };

  useLayoutEffect(() => {
    const container = containerRef.current;
    const tooltip = tooltipRef.current;
    if (!isVisible || !container || !tooltip) return;

    const rect = container.getBoundingClientRect();
    const height = tooltip.getBoundingClientRect().height;
    const spaceBelow = window.innerHeight - rect.bottom;
    const shouldShowBelow = spaceBelow >= height + UI.TOOLTIP_OFFSET;

    setIsBelow(shouldShowBelow);
    setTooltipPosition({
      top: shouldShowBelow
        ? rect.bottom + UI.TOOLTIP_OFFSET
        : rect.top - UI.TOOLTIP_OFFSET,
      left: rect.left,
    });
  }, [children, isVisible, portalRoot]);

  const handleMouseLeave = () => {
    setIsVisible(false);
  };

  useEffect(() => {
    const handleHideTooltips = () => {
      setIsVisible(false);
    };

    document.addEventListener(HIDE_TOOLTIPS_EVENT, handleHideTooltips);
    return () => {
      document.removeEventListener(HIDE_TOOLTIPS_EVENT, handleHideTooltips);
    };
  }, []);

  return {
    isVisible,
    tooltipPosition,
    isBelow,
    portalRoot,
    containerRef,
    tooltipRef,
    handleMouseEnter,
    handleMouseLeave,
  };
};
