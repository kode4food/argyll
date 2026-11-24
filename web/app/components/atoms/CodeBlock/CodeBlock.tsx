import React, { useMemo } from "react";
import { highlight } from "./highlight";
import styles from "./CodeBlock.module.css";

export interface CodeBlockProps {
  code: string;
  language: string;
}

const CodeBlock: React.FC<CodeBlockProps> = ({ code, language }) => {
  const segments = useMemo(() => highlight(code, language), [code, language]);

  return (
    <pre className={styles.block} data-language={language || undefined}>
      <code>
        {segments.map((segment, idx) =>
          segment.className ? (
            <span key={idx} className={segment.className}>
              {segment.text}
            </span>
          ) : (
            <React.Fragment key={idx}>{segment.text}</React.Fragment>
          )
        )}
      </code>
    </pre>
  );
};

export default CodeBlock;
