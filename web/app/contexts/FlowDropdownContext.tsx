import React, { createContext, useContext } from "react";
import { FlowContext } from "../api";

export interface FlowDropdownContextValue {
  showDropdown: boolean;
  setShowDropdown: React.Dispatch<React.SetStateAction<boolean>>;
  searchTerm: string;
  selectedIndex: number;
  searchInputRef: React.RefObject<HTMLInputElement | null>;
  dropdownRef: React.RefObject<HTMLDivElement | null>;
  filteredFlows: FlowContext[];
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  selectFlow: (flowId: string) => void;
  closeDropdown: () => void;
  selectedFlow: string | null;
  flows: FlowContext[];
}

const FlowDropdownContext = createContext<FlowDropdownContextValue | null>(
  null
);

export const FlowDropdownProvider = ({
  value,
  children,
}: {
  value: FlowDropdownContextValue;
  children: React.ReactNode;
}) => (
  <FlowDropdownContext.Provider value={value}>
    {children}
  </FlowDropdownContext.Provider>
);

export const useFlowDropdownContext = (): FlowDropdownContextValue => {
  const ctx = useContext(FlowDropdownContext);
  if (!ctx) {
    throw new Error(
      "useFlowDropdownContext must be used within FlowDropdownProvider"
    );
  }
  return ctx;
};
