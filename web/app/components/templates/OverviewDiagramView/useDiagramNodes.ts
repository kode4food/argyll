import React, { useCallback } from "react";
import { Node, NodeChange, useNodesState } from "@xyflow/react";
import {
  loadNodePositions,
  NodePositionScope,
  saveNodePositions,
} from "@/utils/nodePositioning";

export const useDiagramNodes = (
  arrangedNodes: Node[],
  positionScope: NodePositionScope
) => {
  const [nodes, setNodes, onNodesChange] = useNodesState(arrangedNodes);

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes);

      const positionChanges = changes.filter(
        (change) => change.type === "position" && change.dragging === false
      );
      if (positionChanges.length > 0) {
        setTimeout(() => {
          setNodes((currentNodes) => {
            saveNodePositions(currentNodes, positionScope);
            return currentNodes;
          });
        }, 0);
      }
    },
    [onNodesChange, setNodes, positionScope]
  );

  // A Flow's snapshot copies these, so they must hold every placed Step
  React.useEffect(() => {
    const saved = loadNodePositions(positionScope);
    const unsaved = arrangedNodes.filter((node) => !saved[node.id]);
    if (unsaved.length > 0) {
      saveNodePositions(unsaved, positionScope);
    }
  }, [arrangedNodes, positionScope]);

  React.useEffect(() => {
    setNodes((currentNodes) => {
      if (currentNodes.length === 0) {
        return arrangedNodes;
      }

      const nodeMap = new Map(currentNodes.map((n) => [n.id, n]));
      let hasChanges = currentNodes.length !== arrangedNodes.length;

      const nextNodes = arrangedNodes.map((newNode) => {
        const oldNode = nodeMap.get(newNode.id);
        if (oldNode) {
          if (oldNode.data === newNode.data && oldNode.type === newNode.type) {
            return oldNode;
          }
          hasChanges = true;
          return {
            ...oldNode,
            data: newNode.data,
            type: newNode.type,
          };
        }
        hasChanges = true;
        return newNode;
      });

      return hasChanges ? nextNodes : currentNodes;
    });
  }, [arrangedNodes, setNodes]);

  return { nodes, handleNodesChange };
};
