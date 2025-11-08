import { getProgressIcon, getProgressIconClass } from "./progressUtils";
import {
  Clock,
  Loader2,
  CheckCircle,
  XCircle,
  MinusCircle,
} from "lucide-react";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";

describe("progressUtils", () => {
  describe("getProgressIcon", () => {
    test("returns Clock for pending status", () => {
      expect(getProgressIcon("pending")).toBe(Clock);
    });

    test("returns Loader2 for active status", () => {
      expect(getProgressIcon("active")).toBe(Loader2);
    });

    test("returns CheckCircle for completed status", () => {
      expect(getProgressIcon("completed")).toBe(CheckCircle);
    });

    test("returns XCircle for failed status", () => {
      expect(getProgressIcon("failed")).toBe(XCircle);
    });

    test("returns MinusCircle for skipped status", () => {
      expect(getProgressIcon("skipped")).toBe(MinusCircle);
    });

    test("returns Clock for invalid status", () => {
      expect(getProgressIcon("invalid" as StepProgressStatus)).toBe(Clock);
    });
  });

  describe("getProgressIconClass", () => {
    test("returns pending for pending status", () => {
      expect(getProgressIconClass("pending")).toBe("pending");
    });

    test("returns active for active status", () => {
      expect(getProgressIconClass("active")).toBe("active");
    });

    test("returns completed for completed status", () => {
      expect(getProgressIconClass("completed")).toBe("completed");
    });

    test("returns failed for failed status", () => {
      expect(getProgressIconClass("failed")).toBe("failed");
    });

    test("returns skipped for skipped status", () => {
      expect(getProgressIconClass("skipped")).toBe("skipped");
    });

    test("returns pending for null status", () => {
      expect(getProgressIconClass(null)).toBe("pending");
    });

    test("returns pending for undefined status", () => {
      expect(getProgressIconClass(undefined)).toBe("pending");
    });
  });
});
