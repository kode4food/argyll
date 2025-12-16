import { useMemo, useEffect } from "react";
import { Node } from "@xyflow/react";
import { Step, FlowContext } from "@/app/api";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./diagramUtils";
import { saveNodePositions } from "./nodePositioning";

export function useOverviewAutoLayout(
  visibleSteps: Step[],
  flowData: FlowContext | null,
  arrangedNodes: Node[]
) {
  const overviewPlan = useMemo(() => {
    if (!shouldApplyAutoLayout(flowData, visibleSteps)) {
      return null;
    }
    return generateOverviewPlan(visibleSteps);
  }, [visibleSteps, flowData]);

  useEffect(() => {
    if (!flowData && overviewPlan && arrangedNodes.length > 0) {
      saveNodePositions(arrangedNodes);
    }
  }, [arrangedNodes, flowData, overviewPlan]);

  return { overviewPlan };
}
