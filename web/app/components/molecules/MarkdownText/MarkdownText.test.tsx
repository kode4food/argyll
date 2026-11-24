import { render, screen } from "@testing-library/react";
import MarkdownText from "./MarkdownText";

const renderMarkdown = (source: string) =>
  render(<MarkdownText source={source} />).container;

describe("MarkdownText", () => {
  test("renders headings and inline styles", () => {
    const container = renderMarkdown(
      "# Title\n\nSome **bold**, *em*, ~~gone~~ and `code`\\*"
    );
    expect(screen.getByRole("heading", { name: "Title" })).toBeInTheDocument();
    expect(container.querySelector("strong")).toHaveTextContent("bold");
    expect(container.querySelector("em")).toHaveTextContent("em");
    expect(container.querySelector("del")).toHaveTextContent("gone");
    expect(container.querySelector("code")).toHaveTextContent("code");
    expect(container.querySelector("p")?.textContent).toMatch(/code\*$/);
  });

  test("renders lists, tasks and quotes", () => {
    const container = renderMarkdown(
      "- one\n- [x] done\n\n1. first\n\n> quoted\n\n---"
    );
    expect(container.querySelectorAll("ul > li")).toHaveLength(2);
    expect(screen.getByRole("checkbox")).toBeChecked();
    expect(container.querySelector("ol > li")).toHaveTextContent("first");
    expect(container.querySelector("blockquote")).toHaveTextContent("quoted");
    expect(container.querySelector("hr")).toBeInTheDocument();
  });

  test("highlights fenced code", () => {
    const container = renderMarkdown('```json\n{"a": 1}\n```\n\n    plain');
    const blocks = container.querySelectorAll("pre");
    expect(blocks[0]).toHaveAttribute("data-language", "json");
    expect(blocks[0].querySelector(".tok-number")).toHaveTextContent("1");
    expect(blocks[1]).toHaveTextContent("plain");
  });

  test("renders tables", () => {
    const container = renderMarkdown("| A | B |\n|---|---|\n| 1 | 2 |");
    expect(container.querySelectorAll("th")).toHaveLength(2);
    expect(container.querySelectorAll("tbody td")[1]).toHaveTextContent("2");
  });

  test("links only safe URLs", () => {
    renderMarkdown(
      "[docs](https://example.com) [bad](javascript:alert(1)) <https://a.io>"
    );
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute(
      "href",
      "https://example.com"
    );
    expect(screen.queryByRole("link", { name: "bad" })).toBeNull();
    expect(screen.getByText("bad")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "https://a.io" })).toHaveAttribute(
      "rel",
      "noopener noreferrer"
    );
  });

  test("shows raw HTML as text", () => {
    const container = renderMarkdown("<div>hi</div>\n\n![alt](x.png)");
    expect(container.querySelectorAll("div")).toHaveLength(1);
    expect(container.textContent).toContain("<div>hi</div>");
    expect(container.querySelector("img")).toBeNull();
  });
});
