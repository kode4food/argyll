import { useMemo } from "react";
import dagre from "@dagrejs/dagre";
import { Node, Edge } from "@xyflow/react";
import { ExecutionPlan, AttributeRole } from "@/app/api";
import { STEP_LAYOUT } from "@/constants/layout";

interface LayoutConfig {
  rankdir?: "TB" | "BT" | "LR" | "RL";
  nodeWidth?: number;
  rankSep?: number;
  nodeSep?: number;
}

type StepAttributeSpec = {
  role?: AttributeRole;
};

interface StepNodeData {
  step?: {
    attributes?: Record<string, StepAttributeSpec>;
  };
}

const DEFAULT_CONFIG = {
  rankdir: "LR" as const,
  nodeWidth: 320,
  rankSep: 50,
  nodeSep: 15,
};

const getStepAttributes = (
  node: Node
): Record<string, StepAttributeSpec> | undefined => {
  const data = node.data as StepNodeData | undefined;
  return data?.step?.attributes;
};

const calculateNodeHeight = (node: Node): number => {
  const attributes = getStepAttributes(node);
  if (!attributes) {
    return STEP_LAYOUT.WIDGET_BASE_HEIGHT;
  }

  const requiredCount = Object.values(attributes).filter(
    (spec) => spec?.role === AttributeRole.Required
  ).length;
  const optionalCount = Object.values(attributes).filter(
    (spec) => spec?.role === AttributeRole.Optional
  ).length;
  const outputCount = Object.values(attributes).filter(
    (spec) => spec?.role === AttributeRole.Output
  ).length;

  const calculateSectionHeight = (argCount: number): number => {
    if (argCount === 0) return 0;
    return STEP_LAYOUT.SECTION_HEIGHT + argCount * STEP_LAYOUT.ARG_LINE_HEIGHT;
  };

  return (
    STEP_LAYOUT.WIDGET_BASE_HEIGHT +
    calculateSectionHeight(requiredCount) +
    calculateSectionHeight(optionalCount) +
    calculateSectionHeight(outputCount)
  );
};

export const useAutoLayout = (
  nodes: Node[],
  _edges: Edge[],
  plan: ExecutionPlan | null | undefined,
  config: LayoutConfig = {}
): Node[] => {
  return useMemo(() => {
    if (!plan || nodes.length === 0) {
      return nodes;
    }

    const layoutConfig = { ...DEFAULT_CONFIG, ...config };

    const graph = new dagre.graphlib.Graph();
    graph.setDefaultEdgeLabel(() => ({}));

    graph.setGraph({
      rankdir: layoutConfig.rankdir,
      ranksep: layoutConfig.rankSep,
      nodesep: layoutConfig.nodeSep,
      marginx: 20,
      marginy: 20,
    });

    nodes.forEach((node) => {
      const actualHeight = calculateNodeHeight(node);
      graph.setNode(node.id, {
        width: layoutConfig.nodeWidth,
        height: actualHeight,
      });
    });

    if (plan.attributes) {
      Object.entries(plan.attributes).forEach(([_attrName, deps]) => {
        if (!deps) return;

        deps.providers?.forEach((providerId) => {
          deps.consumers?.forEach((consumerId) => {
            if (
              nodes.some((n) => n.id === providerId) &&
              nodes.some((n) => n.id === consumerId)
            ) {
              graph.setEdge(providerId, consumerId);
            }
          });
        });
      });
    }

    dagre.layout(graph);

    return nodes.map((node) => {
      const nodeWithPosition = graph.node(node.id);

      if (!nodeWithPosition) {
        return node;
      }

      const actualHeight = calculateNodeHeight(node);

      return {
        ...node,
        position: {
          x: nodeWithPosition.x - layoutConfig.nodeWidth / 2,
          y: nodeWithPosition.y - actualHeight / 2,
        },
      };
    });
  }, [nodes, plan, config]);
};
