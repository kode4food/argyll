import { useEffect } from "react";
import { useReactFlow } from "@xyflow/react";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
import { useFitView } from "@/app/hooks/useFitView";

export const useViewportRestore = (viewportKey: string, nodes: unknown[]) => {
  const reactFlowInstance = useReactFlow();
  const fitView = useFitView();
  const {
    handleViewportChange,
    shouldFitView,
    savedViewport,
    markRestored,
    markFitApplied,
  } = useDiagramViewport(viewportKey);

  useEffect(() => {
    if (savedViewport && reactFlowInstance) {
      reactFlowInstance.setViewport(savedViewport);
      requestAnimationFrame(() => markRestored());
    }
  }, [reactFlowInstance, savedViewport, markRestored]);

  useEffect(() => {
    if (!shouldFitView || !reactFlowInstance || nodes.length === 0) {
      return;
    }

    let frameA = 0;
    let frameB = 0;

    frameA = requestAnimationFrame(() => {
      frameB = requestAnimationFrame(() => {
        fitView();
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
  }, [fitView, shouldFitView, nodes, markFitApplied]);

  return handleViewportChange;
};
