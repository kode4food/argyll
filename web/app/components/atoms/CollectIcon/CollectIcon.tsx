import React from "react";
import styles from "./CollectIcon.module.css";

export interface CollectIconProps {
  collect: string;
  className?: string;
  title?: string;
}

const CollectIcon: React.FC<CollectIconProps> = ({
  collect,
  className,
  title,
}) => (
  <span
    className={[styles.icon, className].filter(Boolean).join(" ")}
    title={title}
    style={{
      maskImage: `url(/icons/collect-${collect}.svg)`,
      WebkitMaskImage: `url(/icons/collect-${collect}.svg)`,
    }}
  />
);

export default CollectIcon;
