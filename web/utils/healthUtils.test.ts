import { getHealthIconClass } from "./healthUtils";
import { HealthStatus } from "@/app/api";

describe("healthUtils", () => {
  describe("getHealthIconClass", () => {
    test("returns healthy class for healthy status", () => {
      expect(getHealthIconClass("healthy")).toBe("healthy");
    });

    test("returns unhealthy class for unhealthy status", () => {
      expect(getHealthIconClass("unhealthy")).toBe("unhealthy");
    });

    test("returns unconfigured class for unconfigured status", () => {
      expect(getHealthIconClass("unconfigured")).toBe("unconfigured");
    });

    test("returns unknown class for unknown status", () => {
      expect(getHealthIconClass("unknown")).toBe("unknown");
    });

    test("returns unknown class for invalid status", () => {
      expect(getHealthIconClass("invalid" as HealthStatus)).toBe("unknown");
    });

    test("ignores stepType parameter", () => {
      expect(getHealthIconClass("healthy", "sync")).toBe("healthy");
      expect(getHealthIconClass("unhealthy", "async")).toBe("unhealthy");
    });
  });
});
