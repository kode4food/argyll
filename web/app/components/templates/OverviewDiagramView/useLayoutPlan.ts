import { useMemo, useEffect } from "react";
import { Node } from "@xyflow/react";
import { Step } from "@/app/api";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./layoutUtils";
import { NodePositionScope, saveNodePositions } from "@/utils/nodePositioning";

export function useLayoutPlan(
  visibleSteps: Step[],
  arrangedNodes: Node[],
  scope?: NodePositionScope
) {
  const plan = useMemo(() => {
    if (!shouldApplyAutoLayout(visibleSteps, scope)) {
      return null;
    }
    return generateOverviewPlan(visibleSteps);
  }, [visibleSteps, scope]);

  useEffect(() => {
    if (plan && arrangedNodes.length > 0) {
      saveNodePositions(arrangedNodes, scope);
    }
  }, [arrangedNodes, plan, scope]);

  return { plan };
}
