import {
  getArgIcon,
  getStepTypeIcon,
  IconStepTypeAsync,
  IconStepTypeFlow,
  IconStepTypeScript,
  IconStepTypeSync,
} from "./iconRegistry";

describe("iconRegistry", () => {
  describe("getArgIcon", () => {
    test('returns ArrowRight icon for "required" type', () => {
      const result = getArgIcon("required");
      expect(result.Icon).toBeDefined();
      expect(result.className).toBe("arg-icon input");
    });

    test('returns CircleHelp icon for "optional" type', () => {
      const result = getArgIcon("optional");
      expect(result.Icon).toBeDefined();
      expect(result.className).toBe("arg-icon optional");
    });

    test('returns Lock icon for "const" type', () => {
      const result = getArgIcon("const");
      expect(result.Icon).toBeDefined();
      expect(result.className).toBe("arg-icon const");
    });

    test('returns ArrowLeft icon for "output" type', () => {
      const result = getArgIcon("output");
      expect(result.Icon).toBeDefined();
      expect(result.className).toBe("arg-icon output");
    });
  });

  describe("getStepTypeIcon", () => {
    test("returns Globe for sync steps", () => {
      expect(getStepTypeIcon("sync")).toBe(IconStepTypeSync);
    });

    test("returns Webhook for async steps", () => {
      expect(getStepTypeIcon("async")).toBe(IconStepTypeAsync);
    });

    test("returns FileCode2 for script steps", () => {
      expect(getStepTypeIcon("script")).toBe(IconStepTypeScript);
    });

    test("returns Workflow for flow steps", () => {
      expect(getStepTypeIcon("flow")).toBe(IconStepTypeFlow);
    });
  });
});
