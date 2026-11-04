import { formatCompactJSON, formatJSON } from "./jsonUtils";

describe("formatJSON", () => {
  test("indents objects and arrays", () => {
    expect(formatJSON({ a: 1, b: [2] })).toBe(
      '{\n  "a": 1,\n  "b": [\n    2\n  ]\n}'
    );
  });

  test("formats scalars as JSON", () => {
    expect(formatJSON(null)).toBe("null");
    expect(formatJSON('say "hi"')).toBe('"say \\"hi\\""');
    expect(formatJSON(3.14)).toBe("3.14");
    expect(formatJSON(false)).toBe("false");
  });

  test("falls back to the string form", () => {
    const circular: Record<string, unknown> = { a: 1 };
    circular.self = circular;
    expect(formatJSON(undefined)).toBe("undefined");
    expect(formatJSON(circular)).toBe("[object Object]");
  });
});

describe("formatCompactJSON", () => {
  test("keeps everything on one line", () => {
    expect(formatCompactJSON({ a: 1, b: [2] })).toBe('{"a":1,"b":[2]}');
  });
});
