import React, { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { FlowContext } from "@/app/api";
import { filterFlowsBySearch } from "./flowSelectorUtils";
import { useEscapeKey } from "@/app/hooks/useEscapeKey";
import { FlowDropdownContextValue } from "@/app/contexts/FlowDropdownContext";

type FlowDropdownState = Omit<
  FlowDropdownContextValue,
  "flowsHasMore" | "flowsLoading" | "loadMoreFlows"
>;

export function useFlowDropdownManagement(
  flows: FlowContext[],
  selectedFlow: string | null
): FlowDropdownState {
  const navigate = useNavigate();
  const [showDropdown, setShowDropdown] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredFlows = filterFlowsBySearch(flows, searchTerm);
  const selectableItems = filteredFlows.map((w) => w.id);

  const closeDropdown = () => {
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    setSelectedIndex(-1);
  };

  const navigateToFlow = (flowId: string) => {
    if (flowId === "Overview") {
      navigate("/");
    } else {
      navigate(`/flow/${flowId}`);
    }
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!showDropdown) return;

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < selectableItems.length - 1 ? prev + 1 : 0
        );
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev > 0 ? prev - 1 : selectableItems.length - 1
        );
        break;
      case "Enter":
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < selectableItems.length) {
          navigateToFlow(selectableItems[selectedIndex]);
        }
        break;
      case "Tab":
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < selectableItems.length) {
          navigateToFlow(selectableItems[selectedIndex]);
        } else {
          setSelectedIndex(0);
        }
        break;
    }
  };

  useEffect(() => {
    if (selectedIndex >= 0 && dropdownRef.current) {
      const selectedElement = dropdownRef.current.children[
        selectedIndex + 1
      ] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({
          behavior: "smooth",
          block: "nearest",
        });
      }
    }
  }, [selectedIndex]);

  useEscapeKey(showDropdown, closeDropdown);

  return {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    selectFlow: navigateToFlow,
    closeDropdown,
    selectedFlow,
    flows,
  };
}
