import React from "react";
import { Handle, Position } from "@xyflow/react";

const HANDLE_VERTICAL_OFFSET_PX = 2;

interface InvisibleHandleProps {
  id: string;
  type: "source" | "target";
  position: Position;
  top: number;
  argName: string;
}

const InvisibleHandle: React.FC<InvisibleHandleProps> = ({
  id,
  type,
  position,
  top,
  argName,
}) => {
  const positionClass =
    position === Position.Left
      ? "invisible-handle-left"
      : "invisible-handle-right";

  return (
    <Handle
      key={`${type}-${argName}`}
      type={type}
      position={position}
      id={id}
      isConnectable={false}
      className={`invisible-handle ${positionClass}`}
      style={{ top: `${top + HANDLE_VERTICAL_OFFSET_PX}px` }}
    />
  );
};

export default InvisibleHandle;
