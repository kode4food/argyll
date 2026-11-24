import React from "react";
import type { SyntaxNode } from "@lezer/common";
import CodeBlock from "@/app/components/atoms/CodeBlock";
import { renderInline } from "./markdownInline";
import styles from "./MarkdownText.module.css";

type BlockRenderer = (
  node: SyntaxNode,
  src: string,
  key: number
) => React.ReactNode;

const MAX_HEADING_LEVEL = 6;

// Chat headings start below the box title's level
const HEADING_OFFSET = 2;

const textOf = (node: SyntaxNode | null, src: string) =>
  node ? src.slice(node.from, node.to) : "";

const heading = (level: number): BlockRenderer => {
  const Tag = `h${Math.min(level + HEADING_OFFSET, MAX_HEADING_LEVEL)}` as
    "h3" | "h4" | "h5" | "h6";
  return (node, src, key) => (
    <Tag key={key} className={styles.heading}>
      {renderInline(node, src)}
    </Tag>
  );
};

const childrenNamed = (node: SyntaxNode, name: string) => {
  const res: SyntaxNode[] = [];
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === name) res.push(child);
  }
  return res;
};

const tableRow = (
  row: SyntaxNode,
  src: string,
  key: number,
  Cell: "th" | "td"
) => (
  <tr key={key}>
    {childrenNamed(row, "TableCell").map((cell, idx) => (
      <Cell key={idx}>{renderInline(cell, src)}</Cell>
    ))}
  </tr>
);

const BLOCKS: Record<string, BlockRenderer> = {
  Paragraph: (node, src, key) => (
    <p key={key} className={styles.paragraph}>
      {renderInline(node, src)}
    </p>
  ),
  ATXHeading1: heading(1),
  ATXHeading2: heading(2),
  ATXHeading3: heading(3),
  ATXHeading4: heading(4),
  ATXHeading5: heading(5),
  ATXHeading6: heading(6),
  SetextHeading1: heading(1),
  SetextHeading2: heading(2),
  BulletList: (node, src, key) => (
    <ul key={key} className={styles.list}>
      {renderBlocks(node, src)}
    </ul>
  ),
  OrderedList: (node, src, key) => (
    <ol key={key} className={styles.list}>
      {renderBlocks(node, src)}
    </ol>
  ),
  ListItem: (node, src, key) => <li key={key}>{renderBlocks(node, src)}</li>,
  Blockquote: (node, src, key) => (
    <blockquote key={key} className={styles.quote}>
      {renderBlocks(node, src)}
    </blockquote>
  ),
  FencedCode: (node, src, key) => (
    <CodeBlock
      key={key}
      code={textOf(node.getChild("CodeText"), src)}
      language={textOf(node.getChild("CodeInfo"), src).trim()}
    />
  ),
  CodeBlock: (node, src, key) => (
    <CodeBlock
      key={key}
      code={textOf(node.getChild("CodeText"), src)}
      language=""
    />
  ),
  HorizontalRule: (_node, _src, key) => (
    <hr key={key} className={styles.rule} />
  ),
  Table: (node, src, key) => (
    <div key={key} className={styles.tableWrap}>
      <table className={styles.table}>
        <thead>
          {childrenNamed(node, "TableHeader").map((row, idx) =>
            tableRow(row, src, idx, "th")
          )}
        </thead>
        <tbody>
          {childrenNamed(node, "TableRow").map((row, idx) =>
            tableRow(row, src, idx, "td")
          )}
        </tbody>
      </table>
    </div>
  ),
};

// Renders the block children of a container, showing unknown blocks such as
// raw HTML as plain text
export const renderBlocks = (
  parent: SyntaxNode,
  src: string
): React.ReactNode[] => {
  const res: React.ReactNode[] = [];
  for (let child = parent.firstChild; child; child = child.nextSibling) {
    if (child.name === "ListMark" || child.name === "QuoteMark") continue;
    if (child.name === "Task") {
      res.push(...renderInline(parent, src, child.from, child.to));
      continue;
    }
    const render = BLOCKS[child.name];
    res.push(
      render ? (
        render(child, src, res.length)
      ) : (
        <p key={res.length} className={styles.paragraph}>
          {textOf(child, src)}
        </p>
      )
    );
  }
  return res;
};
