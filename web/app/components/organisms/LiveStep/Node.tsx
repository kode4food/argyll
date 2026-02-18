import React, { useRef } from "react";
import { Position, NodeProps, useUpdateNodeInternals } from "@xyflow/react";
import { Step, FlowContext, ExecutionResult } from "@/app/api";
import Widget from "./Widget";
import InvisibleHandle from "@/app/components/atoms/InvisibleHandle";
import { useNodeData } from "./useNodeData";
import { useHandlePositions } from "@/app/hooks/useHandlePositions";

interface NodeData {
  step: Step;
  selected?: boolean;
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
  isGoalStep?: boolean;
  isStartingPoint?: boolean;
}

const Node: React.FC<NodeProps> = ({ id, data }) => {
  const nodeData = data as unknown as NodeData;
  const { step, flowData, executions = [], resolvedAttributes = [] } = nodeData;
  const widgetRef = useRef<HTMLDivElement | null>(null);
  const updateNodeInternals = useUpdateNodeInternals();

  const { execution, provenance, satisfied } = useNodeData(
    step,
    flowData || null,
    executions,
    resolvedAttributes
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
          selected={nodeData.selected ?? false}
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
          flowId={flowData?.id}
        />
      </div>
    </div>
  );
};

export default React.memo(Node);
