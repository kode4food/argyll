import { getProgressIcon } from "./progressUtils";
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
});
