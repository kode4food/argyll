import {
  formatScriptPreview,
  formatScriptForTooltip,
  stripLuaReturn,
} from "./stepFooterUtils";
import { SCRIPT_LANGUAGE_JPATH, SCRIPT_LANGUAGE_LUA } from "@/app/api";

describe("stepFooterUtils", () => {
  describe("stripLuaReturn", () => {
    it("drops a leading return from a Lua script", () => {
      expect(stripLuaReturn("return x > 10", SCRIPT_LANGUAGE_LUA)).toBe(
        "x > 10"
      );
    });

    it("drops a leading return that is indented", () => {
      expect(stripLuaReturn("  return x > 10", SCRIPT_LANGUAGE_LUA)).toBe(
        "x > 10"
      );
    });

    it("keeps a return that is not the first statement", () => {
      const script = "local y = x * 2\nreturn y > 10";
      expect(stripLuaReturn(script, SCRIPT_LANGUAGE_LUA)).toBe(script);
    });

    it("keeps an identifier that merely starts with return", () => {
      expect(stripLuaReturn("returned > 10", SCRIPT_LANGUAGE_LUA)).toBe(
        "returned > 10"
      );
    });

    it("keeps a bare return", () => {
      expect(stripLuaReturn("return", SCRIPT_LANGUAGE_LUA)).toBe("return");
    });

    it("leaves other languages alone", () => {
      expect(stripLuaReturn("return x", SCRIPT_LANGUAGE_JPATH)).toBe(
        "return x"
      );
      expect(stripLuaReturn("return x")).toBe("return x");
    });
  });

  describe("formatScriptPreview", () => {
    it("replaces newlines with spaces", () => {
      const script = "line1\nline2\nline3";
      expect(formatScriptPreview(script)).toBe("line1 line2 line3");
    });

    it("handles single line scripts", () => {
      const script = "const x = 1;";
      expect(formatScriptPreview(script)).toBe("const x = 1;");
    });

    it("collapses empty lines to a single space", () => {
      const script = "line1\n\nline3";
      expect(formatScriptPreview(script)).toBe("line1 line3");
    });

    it("collapses runs of whitespace, tabs included", () => {
      const script = "line1\n\n\n\tline4     end";
      expect(formatScriptPreview(script)).toBe("line1 line4 end");
    });

    it("trims leading and trailing whitespace", () => {
      expect(formatScriptPreview("  x = 1\n")).toBe("x = 1");
    });

    it("drops a leading return from a Lua script", () => {
      const script = "return {\n  result = 1\n}";
      expect(formatScriptPreview(script, SCRIPT_LANGUAGE_LUA)).toBe(
        "{ result = 1 }"
      );
    });

    // a generated QBE selector arrives joined on " ||\n" and " &&\n    "
    it("collapses a multi-line JPath selector", () => {
      const script =
        '$.tags[?@=="domain:payments"] &&\n' +
        '    $.tags[?@=="tier:gold"] ||\n' +
        '$.tags[?@=="domain:risk"]';
      expect(formatScriptPreview(script, SCRIPT_LANGUAGE_JPATH)).toBe(
        '$.tags[?@=="domain:payments"] && $.tags[?@=="tier:gold"] || ' +
          '$.tags[?@=="domain:risk"]'
      );
    });
  });

  describe("formatScriptForTooltip", () => {
    it("returns preview with first N lines", () => {
      const script = "line1\nline2\nline3\nline4\nline5\nline6";
      const result = formatScriptForTooltip(script, 3);
      expect(result.preview).toBe("line1\nline2\nline3");
      expect(result.lineCount).toBe(6);
    });

    it("handles scripts shorter than maxLines", () => {
      const script = "line1\nline2";
      const result = formatScriptForTooltip(script, 5);
      expect(result.preview).toBe("line1\nline2");
      expect(result.lineCount).toBe(2);
    });

    it("uses default maxLines of 5", () => {
      const script = "line1\nline2\nline3\nline4\nline5\nline6\nline7";
      const result = formatScriptForTooltip(script);
      expect(result.preview).toBe("line1\nline2\nline3\nline4\nline5");
      expect(result.lineCount).toBe(7);
    });

    it("handles single line scripts", () => {
      const script = "single line";
      const result = formatScriptForTooltip(script, 5);
      expect(result.preview).toBe("single line");
      expect(result.lineCount).toBe(1);
    });

    it("handles empty scripts", () => {
      const script = "";
      const result = formatScriptForTooltip(script, 5);
      expect(result.preview).toBe("");
      expect(result.lineCount).toBe(1);
    });

    it("keeps a leading return, which the tooltip shows as written", () => {
      const script = "return x > 10";
      const result = formatScriptForTooltip(script, 5);
      expect(result.preview).toBe("return x > 10");
      expect(result.lineCount).toBe(1);
    });
  });
});
