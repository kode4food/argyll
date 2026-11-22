import React from "react";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import useDropdown from "@/app/hooks/useDropdown";

export const useComboInput = (
  suggestions: readonly string[],
  value: string,
  onChange: (value: string) => void
) => {
  // Typing narrows the list, while the trigger and arrow keys show all of it
  const [isFiltering, setIsFiltering] = React.useState(false);
  const needle = value.toLowerCase();
  const shown = isFiltering
    ? suggestions.filter((s) => s.toLowerCase().includes(needle))
    : suggestions;
  const options = shown.map((s) => ({ value: s }));
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    listRef,
    handleKeyDown,
    selectValue,
  } = useDropdown(options, value, onChange);

  React.useEffect(() => {
    if (!open) setIsFiltering(false);
  }, [open]);

  const handleInputChange = (next: string) => {
    onChange(next);
    setIsFiltering(true);
    setOpen(true);
    setHighlightedIndex(0);
  };

  const handleComboKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === " ") return;
    handleKeyDown(e);
  };

  const handleTriggerClick = () => {
    setIsFiltering(false);
    setOpen((o) => !o);
  };

  // Fixed against the viewport so a scrolling ancestor cannot clip it
  useAnchoredList({
    isOpen: open,
    anchorRef: wrapperRef,
    listRef,
    shouldMatchWidth: true,
  });

  return {
    open,
    shown,
    listRef,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleInputChange,
    handleComboKeyDown,
    handleTriggerClick,
    selectValue,
  };
};
