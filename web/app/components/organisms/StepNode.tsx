import React, { useRef, useCallback } from "react";
import { Position, NodeProps } from "@xyflow/react";
import { Step, FlowContext, ExecutionResult } from "../../api";
import StepWidget from "./StepWidget";
import InvisibleHandle from "../atoms/InvisibleHandle";
import { useDiagramSelection } from "../../contexts/DiagramSelectionContext";
import { useStepNodeData } from "./StepNode/useStepNodeData";
import { useHandlePositions } from "./StepNode/useHandlePositions";

interface StepNodeData {
  step: Step;
  selected: boolean;
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
  isGoalStep?: boolean;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  isStartingPoint?: boolean;
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  disableEdit?: boolean;
}

const StepNode: React.FC<NodeProps> = ({ data }) => {
  const nodeData = data as unknown as StepNodeData;
  const {
    step,
    flowData,
    executions = [],
    resolvedAttributes = [],
    onStepClick,
  } = nodeData;
  const { setGoalSteps } = useDiagramSelection();
  const stepWidgetRef = useRef<HTMLDivElement | null>(null);

  // Memoize the click handler to prevent unnecessary re-renders
  const handleClick = useCallback(
    (event: React.MouseEvent) => {
      const additive = event.ctrlKey || event.metaKey;
      if (onStepClick) {
        onStepClick(step.id, { additive });
      } else if (!additive) {
        setGoalSteps([step.id]);
      }
    },
    [onStepClick, setGoalSteps, step.id]
  );

  // Extract derived data from custom hooks
  const { execution, provenance, satisfied } = useStepNodeData(
    step,
    flowData || null,
    executions,
    resolvedAttributes
  );

  const { allHandles } = useHandlePositions(step, stepWidgetRef);

  return (
    <div>
      {allHandles.map((handle) => (
        <InvisibleHandle
          key={handle.id}
          id={handle.id}
          type={handle.handleType === "output" ? "source" : "target"}
          position={
            handle.handleType === "output" ? Position.Right : Position.Left
          }
          top={handle.top}
          argName={handle.argName}
        />
      ))}

      <div ref={stepWidgetRef}>
        <StepWidget
          step={step}
          selected={nodeData.selected}
          onClick={handleClick}
          mode="diagram"
          className={[
            nodeData.isGoalStep && "goal",
            nodeData.isStartingPoint && "start-point",
          ]
            .filter(Boolean)
            .join(" ")}
          execution={execution}
          satisfiedArgs={satisfied}
          attributeProvenance={provenance}
          attributeValues={flowData?.state}
          isInPreviewPlan={nodeData.isInPreviewPlan}
          isPreviewMode={nodeData.isPreviewMode}
          flowId={flowData?.id}
          diagramContainerRef={nodeData.diagramContainerRef}
          disableEdit={nodeData.disableEdit}
        />
      </div>
    </div>
  );
};

export default React.memo(StepNode);
