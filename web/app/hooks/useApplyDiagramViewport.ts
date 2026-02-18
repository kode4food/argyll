import { useEffect } from "react";
import { ReactFlowInstance, Viewport } from "@xyflow/react";

interface UseApplyDiagramViewportProps {
  fitPadding: number;
  markFitApplied: () => void;
  markRestored: () => void;
  nodeCount: number;
  reactFlowInstance: Pick<ReactFlowInstance, "fitView" | "setViewport">;
  savedViewport: Viewport | null;
  shouldFitView: boolean;
}

export function useApplyDiagramViewport({
  fitPadding,
  markFitApplied,
  markRestored,
  nodeCount,
  reactFlowInstance,
  savedViewport,
  shouldFitView,
}: UseApplyDiagramViewportProps) {
  useEffect(() => {
    if (!savedViewport) {
      return;
    }

    reactFlowInstance.setViewport(savedViewport);
    requestAnimationFrame(() => markRestored());
  }, [markRestored, reactFlowInstance, savedViewport]);

  useEffect(() => {
    if (!shouldFitView || nodeCount === 0) {
      return;
    }

    let frameA = 0;
    let frameB = 0;

    frameA = requestAnimationFrame(() => {
      frameB = requestAnimationFrame(() => {
        reactFlowInstance.fitView({
          padding: fitPadding,
        });
        markFitApplied();
      });
    });

    return () => {
      if (frameA) {
        cancelAnimationFrame(frameA);
      }
      if (frameB) {
        cancelAnimationFrame(frameB);
      }
    };
  }, [fitPadding, markFitApplied, nodeCount, reactFlowInstance, shouldFitView]);
}
