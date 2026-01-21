import { formatScriptPreview, formatScriptForTooltip } from "./stepFooterUtils";

describe("stepFooterUtils", () => {
  describe("formatScriptPreview", () => {
    it("replaces newlines with spaces", () => {
      const script = "line1\nline2\nline3";
      expect(formatScriptPreview(script)).toBe("line1 line2 line3");
    });

    it("handles single line scripts", () => {
      const script = "const x = 1;";
      expect(formatScriptPreview(script)).toBe("const x = 1;");
    });

    it("handles empty lines", () => {
      const script = "line1\n\nline3";
      expect(formatScriptPreview(script)).toBe("line1  line3");
    });

    it("handles scripts with multiple consecutive newlines", () => {
      const script = "line1\n\n\nline4";
      expect(formatScriptPreview(script)).toBe("line1   line4");
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
  });
});
