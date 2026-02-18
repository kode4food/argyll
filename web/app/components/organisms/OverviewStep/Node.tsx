import React, { useRef, useCallback } from "react";
import { Position, NodeProps, useUpdateNodeInternals } from "@xyflow/react";
import { Step } from "@/app/api";
import Widget from "./Widget";
import InvisibleHandle from "@/app/components/atoms/InvisibleHandle";
import { useDiagramSelection } from "@/app/contexts/DiagramSelectionContext";
import { useHandlePositions } from "@/app/hooks/useHandlePositions";

interface NodeData {
  step: Step;
  selected: boolean;
  isGoalStep?: boolean;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  isStartingPoint?: boolean;
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  disableEdit?: boolean;
}

const Node: React.FC<NodeProps> = ({ id, data }) => {
  const nodeData = data as unknown as NodeData;
  const { step, onStepClick } = nodeData;
  const { setGoalSteps } = useDiagramSelection();
  const widgetRef = useRef<HTMLDivElement | null>(null);
  const updateNodeInternals = useUpdateNodeInternals();

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

  const { allHandles } = useHandlePositions(step, widgetRef);

  React.useEffect(() => {
    updateNodeInternals(id);
  }, [allHandles, id, updateNodeInternals]);

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

      <div ref={widgetRef}>
        <Widget
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
          isInPreviewPlan={nodeData.isInPreviewPlan}
          isPreviewMode={nodeData.isPreviewMode}
          diagramContainerRef={nodeData.diagramContainerRef}
          disableEdit={nodeData.disableEdit}
        />
      </div>
    </div>
  );
};

export default React.memo(Node);
