import { render } from "@testing-library/react";
import CodeBlock from "./CodeBlock";

describe("CodeBlock", () => {
  test("highlights a known language", () => {
    const { container } = render(
      <CodeBlock code={'{"a": 1}\n"b"'} language="json" />
    );
    expect(container.textContent).toBe('{"a": 1}\n"b"');
    expect(container.querySelector(".tok-number")).toHaveTextContent("1");
    expect(container.querySelector("pre")).toHaveAttribute(
      "data-language",
      "json"
    );
  });

  test("renders unknown languages as plain text", () => {
    const { container } = render(<CodeBlock code="x := 1" language="" />);
    expect(container.textContent).toBe("x := 1");
    expect(container.querySelector("span")).toBeNull();
  });

  test("matches language names without case", () => {
    const { container } = render(
      <CodeBlock code="local x = 1" language="Lua" />
    );
    expect(container.querySelector(".tok-keyword")).toHaveTextContent("local");
  });
});
