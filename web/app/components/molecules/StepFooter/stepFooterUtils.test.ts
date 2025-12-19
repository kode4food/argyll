import {
  formatScriptPreview,
  getScriptIcon,
  getHttpIcon,
  getSkipReason,
  formatScriptForTooltip,
} from "./stepFooterUtils";
import { SCRIPT_LANGUAGE_ALE } from "../../../api";
import { FileCode2, Code2, Webhook, Globe } from "lucide-react";

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

  describe("getScriptIcon", () => {
    it("returns FileCode2 for ALE language", () => {
      expect(getScriptIcon(SCRIPT_LANGUAGE_ALE)).toBe(FileCode2);
    });

    it("returns Code2 for other languages", () => {
      expect(getScriptIcon("python")).toBe(Code2);
      expect(getScriptIcon("javascript")).toBe(Code2);
      expect(getScriptIcon("")).toBe(Code2);
    });
  });

  describe("getHttpIcon", () => {
    it("returns Webhook for async steps", () => {
      expect(getHttpIcon("async")).toBe(Webhook);
    });

    it("returns Globe for sync steps", () => {
      expect(getHttpIcon("sync")).toBe(Globe);
    });

    it("returns Globe for other step types", () => {
      expect(getHttpIcon("custom")).toBe(Globe);
      expect(getHttpIcon("")).toBe(Globe);
    });
  });

  describe("getSkipReason", () => {
    it("returns predicate reason when step has predicate", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
        predicate: "x > 0",
      };
      expect(getSkipReason(step as any)).toBe(
        "Step skipped because predicate evaluated to false"
      );
    });

    it("returns input unavailable reason when step has no predicate", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
      };
      expect(getSkipReason(step as any)).toBe(
        "Step skipped because required inputs are unavailable due to failed or skipped upstream steps"
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
  });
});
