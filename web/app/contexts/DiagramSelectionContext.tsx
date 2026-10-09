import React, { createContext, useContext } from "react";
import { GoalSets } from "@/app/api";
import { useUI } from "./UIContext";

export interface DiagramSelectionContextValue {
  goalSteps: GoalSets;
  setGoalSteps: (goals: GoalSets) => void;
}

const DiagramSelectionContext =
  createContext<DiagramSelectionContextValue | null>(null);

export const DiagramSelectionProvider = ({
  children,
  value,
}: {
  children: React.ReactNode;
  value?: DiagramSelectionContextValue;
}) => {
  const ui = useUI();
  const contextValue =
    value ||
    ({
      goalSteps: ui.goalSteps,
      setGoalSteps: ui.setGoalSteps,
    } satisfies DiagramSelectionContextValue);

  return (
    <DiagramSelectionContext.Provider value={contextValue}>
      {children}
    </DiagramSelectionContext.Provider>
  );
};

export const useDiagramSelection = (): DiagramSelectionContextValue => {
  const ctx = useContext(DiagramSelectionContext);
  if (!ctx) {
    throw new Error(
      "useDiagramSelection must be used within DiagramSelectionProvider"
    );
  }
  return ctx;
};
