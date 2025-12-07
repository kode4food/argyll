import React, { createContext, useContext } from "react";
import { FlowStatus } from "../api";

interface FlowDropdownContextValue {
  showDropdown: boolean;
  setShowDropdown: (open: boolean) => void;
  searchTerm: string;
  selectedIndex: number;
  searchInputRef: React.RefObject<HTMLInputElement | null>;
  dropdownRef: React.RefObject<HTMLDivElement | null>;
  filteredFlows: { id: string; status: FlowStatus }[];
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  selectFlow: (flowId: string) => void;
  closeDropdown: () => void;
  selectedFlow: string | null;
  flows: { id: string; status: FlowStatus }[];
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
