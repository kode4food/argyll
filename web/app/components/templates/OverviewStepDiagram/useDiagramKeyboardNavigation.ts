import { useMemo, useCallback } from "react";
import { Node } from "@xyflow/react";
import {
  groupNodesByLevel,
  findNextStepInDirection,
} from "./diagramNavigationUtils";

export function useDiagramKeyboardNavigation(
  nodes: Node[],
  activeGoalStepId: string | null,
  handleStepClick: (stepId: string) => void
) {
  const stepsByLevel = useMemo(() => groupNodesByLevel(nodes), [nodes]);

  const findNextStep = useCallback(
    (direction: "up" | "down" | "left" | "right") => {
      return findNextStepInDirection(
        direction,
        activeGoalStepId,
        nodes,
        stepsByLevel
      );
    },
    [activeGoalStepId, nodes, stepsByLevel]
  );

  const handleArrowUp = useCallback(() => {
    const nextStep = findNextStep("up");
    if (nextStep) handleStepClick(nextStep);
  }, [findNextStep, handleStepClick]);

  const handleArrowDown = useCallback(() => {
    const nextStep = findNextStep("down");
    if (nextStep) handleStepClick(nextStep);
  }, [findNextStep, handleStepClick]);

  const handleArrowLeft = useCallback(() => {
    const nextStep = findNextStep("left");
    if (nextStep) handleStepClick(nextStep);
  }, [findNextStep, handleStepClick]);

  const handleArrowRight = useCallback(() => {
    const nextStep = findNextStep("right");
    if (nextStep) handleStepClick(nextStep);
  }, [findNextStep, handleStepClick]);

  const handleEnter = useCallback(() => {
    if (activeGoalStepId) {
      const event = new CustomEvent("openStepEditor", {
        detail: { stepId: activeGoalStepId },
      });
      document.dispatchEvent(event);
    }
  }, [activeGoalStepId]);

  const handleEscape = useCallback(() => {
    const event = new CustomEvent("clearSelection");
    document.dispatchEvent(event);
  }, []);

  return {
    handleArrowUp,
    handleArrowDown,
    handleArrowLeft,
    handleArrowRight,
    handleEnter,
    handleEscape,
  };
}
