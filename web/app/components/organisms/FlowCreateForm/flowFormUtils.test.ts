import {
  buildItemClassName,
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  formatInputValue,
  hasScrollOverflow,
  parseInputValue,
  safeParseState,
  validateJsonString,
} from "./flowFormUtils";

const createScrollableElement = ({
  scrollTop,
  scrollHeight,
  clientHeight,
}: {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
}) => {
  const el = document.createElement("div");
  Object.defineProperty(el, "scrollTop", { value: scrollTop, writable: true });
  Object.defineProperty(el, "scrollHeight", { value: scrollHeight });
  Object.defineProperty(el, "clientHeight", { value: clientHeight });
  return el;
};

describe("flowFormUtils", () => {
  describe("hasScrollOverflow", () => {
    it("returns no overflow state when content fits", () => {
      const el = createScrollableElement({
        scrollTop: 0,
        scrollHeight: 100,
        clientHeight: 100,
      });

      expect(hasScrollOverflow(el)).toEqual({
        hasOverflow: false,
        atTop: true,
        atBottom: true,
      });
    });

    it("detects overflow and middle scroll position", () => {
      const el = createScrollableElement({
        scrollTop: 20,
        scrollHeight: 200,
        clientHeight: 100,
      });

      expect(hasScrollOverflow(el)).toEqual({
        hasOverflow: true,
        atTop: false,
        atBottom: false,
      });
    });

    it("marks bottom when scrolled near the end", () => {
      const el = createScrollableElement({
        scrollTop: 99,
        scrollHeight: 200,
        clientHeight: 100,
      });

      expect(hasScrollOverflow(el)).toEqual({
        hasOverflow: true,
        atTop: false,
        atBottom: true,
      });
    });
  });

  describe("safeParseState", () => {
    it("parses valid JSON strings", () => {
      const parsed = safeParseState('{"foo": "bar"}');
      expect(parsed).toEqual({ foo: "bar" });
    });

    it("returns empty object for non-object JSON", () => {
      expect(safeParseState("null")).toEqual({});
      expect(safeParseState("123")).toEqual({});
      expect(safeParseState('["a"]')).toEqual({});
    });

    it("returns empty object for invalid JSON", () => {
      const parsed = safeParseState("{not-valid");
      expect(parsed).toEqual({});
    });
  });

  describe("validateJsonString", () => {
    it("returns null for valid JSON", () => {
      expect(validateJsonString('{"valid": true}')).toBeNull();
    });

    it("returns an error message for invalid JSON", () => {
      const result = validateJsonString("{invalid");
      expect(result).not.toBeNull();
      expect(typeof result).toBe("string");
      expect((result as string).length).toBeGreaterThan(0);
    });
  });

  describe("buildItemClassName", () => {
    it("combines only truthy class names", () => {
      expect(
        buildItemClassName(true, false, "base", "selected", "disabled")
      ).toBe("base selected");

      expect(
        buildItemClassName(false, true, "base", "selected", "disabled")
      ).toBe("base disabled");

      expect(
        buildItemClassName(false, false, "base", "selected", "disabled")
      ).toBe("base");
    });
  });

  describe("formatInputValue", () => {
    it("returns empty string for nullish values", () => {
      expect(formatInputValue(undefined)).toBe("");
      expect(formatInputValue(null)).toBe("");
    });

    it("returns strings as-is", () => {
      expect(formatInputValue("abc")).toBe("abc");
    });

    it("serializes non-string values", () => {
      expect(formatInputValue(123)).toBe("123");
      expect(formatInputValue(true)).toBe("true");
      expect(formatInputValue({ a: 1 })).toBe('{"a":1}');
    });
  });

  describe("parseInputValue", () => {
    it("returns undefined for empty strings", () => {
      expect(parseInputValue("   ")).toBeUndefined();
    });

    it("parses valid JSON literals", () => {
      expect(parseInputValue("123")).toBe(123);
      expect(parseInputValue("true")).toBe(true);
      expect(parseInputValue('{"a":1}')).toEqual({ a: 1 });
    });

    it("falls back to raw string for non-JSON text", () => {
      expect(parseInputValue("hello")).toBe("hello");
    });
  });

  describe("initial state input value helpers", () => {
    it("builds table values from initial state", () => {
      const values = buildInitialStateInputValues('{"a":1,"b":"text"}', [
        "a",
        "b",
        "c",
      ]);
      expect(values).toEqual({
        a: "1",
        b: "text",
        c: "",
      });
    });

    it("builds initial state from table values", () => {
      const state = buildInitialStateFromInputValues(
        {
          a: "1",
          b: "hello",
          c: "",
          d: '{"x":true}',
        },
        ["a", "b", "c", "d"]
      );

      expect(state).toEqual({
        a: 1,
        b: "hello",
        d: { x: true },
      });
    });
  });
});
