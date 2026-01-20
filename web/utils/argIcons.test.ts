import { getArgIcon } from "./argIcons";
import { ArrowRight, ArrowLeft, CircleHelp, Lock } from "lucide-react";

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

    test('returns Lock icon for "const" type', () => {
      const result = getArgIcon("const");

      expect(result.Icon).toBe(Lock);
      expect(result.className).toBe("arg-icon const");
    });

    test('returns ArrowLeft icon for "output" type', () => {
      const result = getArgIcon("output");

      expect(result.Icon).toBe(ArrowLeft);
      expect(result.className).toBe("arg-icon output");
    });
  });
});
