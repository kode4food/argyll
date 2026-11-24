import React from "react";
import type { SyntaxNode } from "@lezer/common";
import styles from "./MarkdownText.module.css";

type InlineRenderer = (
  node: SyntaxNode,
  src: string,
  key: number
) => React.ReactNode;

const SAFE_HREF = /^(https?:|mailto:)/i;

// Syntax marks carry no content of their own
const MARKS = new Set([
  "CodeMark",
  "EmphasisMark",
  "HeaderMark",
  "LinkMark",
  "ListMark",
  "QuoteMark",
  "StrikethroughMark",
  "TaskMarker",
  "LinkTitle",
  "LinkLabel",
]);

const textOf = (node: SyntaxNode, src: string) => src.slice(node.from, node.to);

const linkTo = (href: string, children: React.ReactNode, key: number) =>
  SAFE_HREF.test(href) ? (
    <a key={key} href={href} target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  ) : (
    <React.Fragment key={key}>{children}</React.Fragment>
  );

// Link text sits between the first two LinkMarks: [text](url)
const linkLabel = (node: SyntaxNode, src: string) => {
  const open = node.getChild("LinkMark");
  const close = open?.nextSibling;
  if (!open || !close) return textOf(node, src);
  return renderInline(node, src, open.to, close.from);
};

const INLINE: Record<string, InlineRenderer> = {
  Emphasis: (node, src, key) => <em key={key}>{renderInline(node, src)}</em>,
  StrongEmphasis: (node, src, key) => (
    <strong key={key}>{renderInline(node, src)}</strong>
  ),
  Strikethrough: (node, src, key) => (
    <del key={key}>{renderInline(node, src)}</del>
  ),
  InlineCode: (node, src, key) => (
    <code key={key} className={styles.inlineCode}>
      {renderInline(node, src)}
    </code>
  ),
  Link: (node, src, key) => {
    const url = node.getChild("URL");
    const label = linkLabel(node, src);
    return url ? linkTo(textOf(url, src), label, key) : label;
  },
  Autolink: (node, src, key) => {
    const url = node.getChild("URL");
    const href = url ? textOf(url, src) : "";
    return linkTo(href, href, key);
  },
  URL: (node, src, key) => linkTo(textOf(node, src), textOf(node, src), key),
  Image: (node, src) => linkLabel(node, src),
  HardBreak: (_node, _src, key) => <br key={key} />,
  Escape: (node, src) => textOf(node, src).slice(1),
  Task: (node, src, key) => (
    <React.Fragment key={key}>
      <input
        type="checkbox"
        disabled
        checked={/\[[xX]\]/.test(textOf(node, src))}
        className={styles.task}
      />
      {renderInline(node, src)}
    </React.Fragment>
  ),
};

// Renders a node's inline content, keeping the source text between children
export const renderInline = (
  parent: SyntaxNode,
  src: string,
  from = parent.from,
  to = parent.to
): React.ReactNode[] => {
  const res: React.ReactNode[] = [];
  let pos = from;
  for (let child = parent.firstChild; child; child = child.nextSibling) {
    if (child.to <= from || child.from >= to) continue;
    if (child.from > pos) res.push(src.slice(pos, child.from));
    pos = child.to;
    if (MARKS.has(child.name)) continue;
    const render = INLINE[child.name];
    res.push(render ? render(child, src, res.length) : textOf(child, src));
  }
  if (pos < to) res.push(src.slice(pos, to));
  return res;
};
