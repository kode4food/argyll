import { getProgressIcon } from "./progressUtils";
import {
  IconProgressActive,
  IconProgressCompleted,
  IconProgressFailed,
  IconProgressPending,
  IconProgressSkipped,
} from "./iconRegistry";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";

describe("progressUtils", () => {
  describe("getProgressIcon", () => {
    test("returns Clock for pending status", () => {
      expect(getProgressIcon("pending")).toBe(IconProgressPending);
    });

    test("returns Loader2 for active status", () => {
      expect(getProgressIcon("active")).toBe(IconProgressActive);
    });

    test("returns CheckCircle for completed status", () => {
      expect(getProgressIcon("completed")).toBe(IconProgressCompleted);
    });

    test("returns XCircle for failed status", () => {
      expect(getProgressIcon("failed")).toBe(IconProgressFailed);
    });

    test("returns MinusCircle for skipped status", () => {
      expect(getProgressIcon("skipped")).toBe(IconProgressSkipped);
    });

    test("returns Clock for invalid status", () => {
      expect(getProgressIcon("invalid" as StepProgressStatus)).toBe(
        IconProgressPending
      );
    });
  });
});
