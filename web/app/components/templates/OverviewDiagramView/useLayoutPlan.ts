import { useMemo } from "react";
import { Step } from "@/app/api";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./layoutUtils";
import { NodePositionScope } from "@/utils/nodePositioning";

export function useLayoutPlan(visibleSteps: Step[], scope?: NodePositionScope) {
  const plan = useMemo(() => {
    if (!shouldApplyAutoLayout(visibleSteps, scope)) {
      return null;
    }
    return generateOverviewPlan(visibleSteps);
  }, [visibleSteps, scope]);

  return { plan };
}
