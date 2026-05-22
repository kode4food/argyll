import React from "react";
import styles from "./SegmentedGroup.module.css";

interface SegmentedGroupProps {
  children: React.ReactNode;
  className?: string;
}

const SegmentedGroup: React.FC<SegmentedGroupProps> = ({
  children,
  className,
}) => {
  const childArray = React.Children.toArray(children).filter(Boolean);
  const content: React.ReactNode[] = [];
  childArray.forEach((child, index) => {
    if (index > 0) {
      content.push(
        <div
          key={`sep-${index}`}
          className={styles.separator}
          aria-hidden="true"
        />
      );
    }
    content.push(child);
  });

  return (
    <div className={[styles.group, className].filter(Boolean).join(" ")}>
      {content}
    </div>
  );
};

export default SegmentedGroup;
