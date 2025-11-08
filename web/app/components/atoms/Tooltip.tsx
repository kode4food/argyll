import React, { useEffect, useRef, useState } from "react";
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
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseEnter = () => {
    if (containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      const tooltipHeight = 120;
      const spaceBelow = window.innerHeight - rect.bottom;

      const shouldShowBelow = spaceBelow >= tooltipHeight + UI.TOOLTIP_OFFSET;

      setShowBelow(shouldShowBelow);
      setTooltipPosition({
        top: shouldShowBelow
          ? rect.bottom + UI.TOOLTIP_OFFSET
          : rect.top - UI.TOOLTIP_OFFSET,
        left: rect.left,
      });
      setIsVisible(true);
    }
  };

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
      className={`${styles.portal} ${isVisible ? styles.visible : ""} ${showBelow ? styles.below : styles.above}`}
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
      {createPortal(tooltip, document.body)}
    </>
  );
};

export default Tooltip;
