import React, { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { UI } from "@/constants/common";
import styles from "./Tooltip.module.css";

interface TooltipProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
}

const Tooltip: React.FC<TooltipProps> = ({ trigger, children }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ top: 0, left: 0 });
  const [showBelow, setShowBelow] = useState(false);
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

    setShowBelow(shouldShowBelow);
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

    document.addEventListener("hideTooltips", handleHideTooltips);
    return () => {
      document.removeEventListener("hideTooltips", handleHideTooltips);
    };
  }, []);

  const tooltip = (
    <div
      ref={tooltipRef}
      className={[
        styles.portal,
        isVisible && styles.visible,
        showBelow ? styles.below : styles.above,
      ]
        .filter(Boolean)
        .join(" ")}
      style={{
        position: "fixed",
        top: tooltipPosition.top,
        left: tooltipPosition.left,
        transform: showBelow ? "translateY(0)" : "translateY(-100%)",
      }}
    >
      {children}
    </div>
  );

  return (
    <>
      <div
        ref={containerRef}
        className={styles.container}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        {trigger}
      </div>
      {createPortal(tooltip, portalRoot)}
    </>
  );
};

export default Tooltip;
