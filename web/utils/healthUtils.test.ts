import { getHealthIconClass, getHealthStatusText } from "./healthUtils";
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

  describe("getHealthStatusText", () => {
    test("returns Healthy for healthy status", () => {
      expect(getHealthStatusText("healthy")).toBe("Healthy");
    });

    test("returns Unhealthy for unhealthy status without error", () => {
      expect(getHealthStatusText("unhealthy")).toBe("Unhealthy");
    });

    test("returns error message for unhealthy status with error", () => {
      expect(getHealthStatusText("unhealthy", "Connection timeout")).toBe(
        "Connection timeout"
      );
    });

    test("returns No health check configured for unconfigured status", () => {
      expect(getHealthStatusText("unconfigured")).toBe(
        "No health check configured"
      );
    });

    test("returns Unknown for unknown status", () => {
      expect(getHealthStatusText("unknown")).toBe("Unknown");
    });

    test("returns Unknown for invalid status", () => {
      expect(getHealthStatusText("invalid" as HealthStatus)).toBe("Unknown");
    });

    test("ignores error parameter for non-unhealthy statuses", () => {
      expect(getHealthStatusText("healthy", "Some error")).toBe("Healthy");
      expect(getHealthStatusText("unconfigured", "Some error")).toBe(
        "No health check configured"
      );
    });
  });
});
