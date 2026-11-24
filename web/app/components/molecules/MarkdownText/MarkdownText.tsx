import React, { useMemo } from "react";
import { GFM, parser } from "@lezer/markdown";
import { renderBlocks } from "./markdownBlocks";
import styles from "./MarkdownText.module.css";

export interface MarkdownTextProps {
  source: string;
}

const MARKDOWN_PARSER = parser.configure(GFM);

const MarkdownText: React.FC<MarkdownTextProps> = ({ source }) => {
  const blocks = useMemo(
    () => renderBlocks(MARKDOWN_PARSER.parse(source).topNode, source),
    [source]
  );
  return <div className={styles.markdown}>{blocks}</div>;
};

export default MarkdownText;
