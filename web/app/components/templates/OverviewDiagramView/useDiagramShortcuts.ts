import React from "react";
import { Node } from "@xyflow/react";
import { useT } from "@/app/i18n";
import { useKeyboardShortcuts } from "@/app/hooks/useKeyboardShortcuts";
import useFocusWithin from "@/app/hooks/useFocusWithin";
import { useKeyboardNavigation } from "./useKeyboardNavigation";

export interface DiagramShortcutsOptions {
  nodes: Node[];
  activeGoalStepId: string | null;
  handleStepClick: (stepId: string) => void;
  diagramContainerRef: React.RefObject<HTMLDivElement | null>;
}

export const useDiagramShortcuts = ({
  nodes,
  activeGoalStepId,
  handleStepClick,
  diagramContainerRef,
}: DiagramShortcutsOptions) => {
  const t = useT();
  const {
    handleArrowUp,
    handleArrowDown,
    handleArrowLeft,
    handleArrowRight,
    handleEnter,
    handleEscape,
  } = useKeyboardNavigation(nodes, activeGoalStepId, handleStepClick);

  const hasDiagramFocus = useFocusWithin(diagramContainerRef);

  useKeyboardShortcuts(
    [
      {
        key: "ArrowUp",
        description: t("keyboardShortcuts.navigateUp"),
        handler: handleArrowUp,
      },
      {
        key: "ArrowDown",
        description: t("keyboardShortcuts.navigateDown"),
        handler: handleArrowDown,
      },
      {
        key: "ArrowLeft",
        description: t("keyboardShortcuts.navigateLeft"),
        handler: handleArrowLeft,
      },
      {
        key: "ArrowRight",
        description: t("keyboardShortcuts.navigateRight"),
        handler: handleArrowRight,
      },
      {
        key: "Enter",
        description: t("keyboardShortcuts.openEditor"),
        handler: handleEnter,
      },
      {
        key: "Escape",
        description: t("keyboardShortcuts.deselectStep"),
        handler: handleEscape,
      },
    ],
    hasDiagramFocus
  );
};
