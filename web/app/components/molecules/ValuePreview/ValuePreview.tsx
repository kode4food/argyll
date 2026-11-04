import React from "react";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import { formatCompactJSON, formatJSON } from "@/utils/jsonUtils";
import styles from "./ValuePreview.module.css";

export interface ValuePreviewProps {
  title: string;
  value: unknown;
}

// A value as compact JSON clipped to two lines, shown in full on hover
const ValuePreview: React.FC<ValuePreviewProps> = ({ title, value }) => (
  <Tooltip
    trigger={<code className={styles.value}>{formatCompactJSON(value)}</code>}
  >
    <TooltipSection title={title} monospace>
      {formatJSON(value)}
    </TooltipSection>
  </Tooltip>
);

export default ValuePreview;
