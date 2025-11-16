import { getArgIcon } from "./argIcons";
import { ArrowRight, ArrowLeft, CircleHelp } from "lucide-react";

describe("argIcons", () => {
  describe("getArgIcon", () => {
    test('returns ArrowRight icon for "required" type', () => {
      const result = getArgIcon("required");

      expect(result.Icon).toBe(ArrowRight);
      expect(result.className).toBe("arg-icon input");
    });

    test('returns CircleHelp icon for "optional" type', () => {
      const result = getArgIcon("optional");

      expect(result.Icon).toBe(CircleHelp);
      expect(result.className).toBe("arg-icon optional");
    });

    test('returns ArrowLeft icon for "output" type', () => {
      const result = getArgIcon("output");

      expect(result.Icon).toBe(ArrowLeft);
      expect(result.className).toBe("arg-icon output");
    });
  });
});
