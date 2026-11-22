import React, { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { FlowSummary } from "@/app/api";
import { filterFlowsBySearch } from "./flowSelectorUtils";
import { useEscapeKey } from "@/app/hooks/useEscapeKey";

const KEY_DIRECTIONS: Record<string, 1 | -1> = { ArrowDown: 1, ArrowUp: -1 };

const stepIndex = (prev: number, direction: 1 | -1, length: number) => {
  if (direction === 1) return prev < length - 1 ? prev + 1 : 0;
  return prev > 0 ? prev - 1 : length - 1;
};

export function useFlowDropdownManagement(
  flows: FlowSummary[],
  selectedFlow: string | null
) {
  const navigate = useNavigate();
  const [showDropdown, setShowDropdown] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [scrollTop, setScrollTop] = useState(0);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredFlows = filterFlowsBySearch(flows, searchTerm);
  const selectableItems = filteredFlows.map((w) => w.id);
  const currentIndex = selectableItems.findIndex((id) => id === selectedFlow);

  const closeDropdown = () => {
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
    setScrollTop(0);
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

    const direction = KEY_DIRECTIONS[e.key];
    if (direction) {
      e.preventDefault();
      setSelectedIndex((prev) =>
        stepIndex(prev, direction, selectableItems.length)
      );
    } else if (e.key === "Enter" || e.key === "Tab") {
      e.preventDefault();
      confirmSelection(e.key === "Tab");
    }
  };

  const confirmSelection = (isTab: boolean) => {
    const hasSelection =
      selectedIndex >= 0 && selectedIndex < selectableItems.length;
    if (hasSelection) {
      navigateToFlow(selectableItems[selectedIndex]);
    } else if (isTab) {
      setSelectedIndex(0);
    } else if (selectableItems.length === 1) {
      navigateToFlow(selectableItems[0]);
    }
  };

  useEffect(() => {
    const list = dropdownRef.current;
    if (!showDropdown || searchTerm || !list || currentIndex < 0) return;
    list.children[currentIndex + 1]?.scrollIntoView({ block: "nearest" });
  }, [showDropdown, searchTerm, currentIndex]);

  useEffect(() => {
    if (selectedIndex < 0) return;
    dropdownRef.current?.children[selectedIndex + 1]?.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
    });
  }, [selectedIndex]);

  useEscapeKey(showDropdown, closeDropdown);

  return {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    setSelectedIndex,
    scrollTop,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    setScrollTop,
    selectFlow: navigateToFlow,
    closeDropdown,
    selectedFlow,
    flows,
  };
}
