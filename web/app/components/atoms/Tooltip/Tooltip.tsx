import React from "react";
import { createPortal } from "react-dom";
import styles from "./Tooltip.module.css";
import { useTooltip } from "./useTooltip";

interface TooltipProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
}

const Tooltip: React.FC<TooltipProps> = ({ trigger, children }) => {
  const {
    isVisible,
    tooltipPosition,
    isBelow,
    portalRoot,
    containerRef,
    tooltipRef,
    handleMouseEnter,
    handleMouseLeave,
  } = useTooltip(children);

  const tooltip = (
    <div
      ref={tooltipRef}
      className={[
        styles.portal,
        isVisible && styles.visible,
        isBelow ? styles.below : styles.above,
      ]
        .filter(Boolean)
        .join(" ")}
      style={{
        position: "fixed",
        top: tooltipPosition.top,
        left: tooltipPosition.left,
        transform: isBelow ? "translateY(0)" : "translateY(-100%)",
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
