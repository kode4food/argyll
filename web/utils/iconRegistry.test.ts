import type { Step } from "@/app/api";
import {
  getArgIcon,
  getStepActionIcon,
  getStepTypeIcon,
  IconActionModeAsync,
  IconActionModeSync,
  IconStepTypeFlow,
  IconStepTypeScript,
  IconStepTypeService,
} from "./iconRegistry";

const serviceStep = (mode: "sync" | "async"): Step => ({
  id: "step",
  name: "Step",
  type: "service",
  attributes: {},
  http: { invoke: { endpoint: "http://test", mode } },
});

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
    test("returns Globe for service steps", () => {
      expect(getStepTypeIcon("service")).toBe(IconStepTypeService);
    });

    test("returns FileCode2 for script steps", () => {
      expect(getStepTypeIcon("script")).toBe(IconStepTypeScript);
    });

    test("returns Workflow for flow steps", () => {
      expect(getStepTypeIcon("flow")).toBe(IconStepTypeFlow);
    });
  });

  describe("getStepActionIcon", () => {
    test("returns the sync icon for a sync Service step", () => {
      expect(getStepActionIcon(serviceStep("sync"))).toBe(IconActionModeSync);
    });

    test("returns the async icon for an async Service step", () => {
      expect(getStepActionIcon(serviceStep("async"))).toBe(IconActionModeAsync);
    });

    test("returns the sync icon when a Service step omits its mode", () => {
      const step = { ...serviceStep("sync"), http: undefined };
      expect(getStepActionIcon(step)).toBe(IconActionModeSync);
    });

    test("returns the type icon for non-Service steps", () => {
      const step: Step = {
        id: "s",
        name: "S",
        type: "script",
        attributes: {},
      };
      expect(getStepActionIcon(step)).toBe(IconStepTypeScript);
    });
  });
});
