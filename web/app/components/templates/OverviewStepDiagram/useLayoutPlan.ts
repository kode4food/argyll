import { useMemo, useEffect } from "react";
import { Node } from "@xyflow/react";
import { Step } from "@/app/api";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./diagramUtils";
import { saveNodePositions } from "@/utils/nodePositioning";

export function useLayoutPlan(visibleSteps: Step[], arrangedNodes: Node[]) {
  const plan = useMemo(() => {
    if (!shouldApplyAutoLayout(visibleSteps)) {
      return null;
    }
    return generateOverviewPlan(visibleSteps);
  }, [visibleSteps]);

  useEffect(() => {
    if (plan && arrangedNodes.length > 0) {
      saveNodePositions(arrangedNodes);
    }
  }, [arrangedNodes, plan]);

  return { plan };
}
