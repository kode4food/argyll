import React, { createContext, useContext } from "react";
import { useUI } from "./UIContext";

export interface DiagramSelectionContextValue {
  selectedStep: string | null;
  setSelectedStep: (id: string | null) => void;
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
      selectedStep: ui.selectedStep ?? null,
      setSelectedStep: ui.setSelectedStep,
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
