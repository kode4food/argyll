import React from "react";
import { Position, NodeProps } from "@xyflow/react";
import { Step } from "@/app/api";
import Widget from "./Widget";
import InvisibleHandle from "@/app/components/atoms/InvisibleHandle";
import { useNodeHandles } from "@/app/hooks/useNodeHandles";
import { useNodeClick } from "./useNodeClick";

interface NodeData {
  step: Step;
  selected: boolean;
  isGoalStep?: boolean;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  isStartingPoint?: boolean;
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const Node: React.FC<NodeProps> = ({ id, data }) => {
  const nodeData = data as unknown as NodeData;
  const { step, onStepClick } = nodeData;
  const handleClick = useNodeClick(step.id, onStepClick);
  const { widgetRef, allHandles, summaryInputTop, summaryOutputTop } =
    useNodeHandles(id, step);

  return (
    <div>
      <InvisibleHandle
        id="summary-input"
        type="target"
        position={Position.Left}
        argName="summary"
        top={summaryInputTop}
      />
      <InvisibleHandle
        id="summary-output"
        type="source"
        position={Position.Right}
        argName="summary"
        top={summaryOutputTop}
      />
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
        />
      </div>
    </div>
  );
};

export default React.memo(Node);
