import React from "react";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import useDropdown from "@/app/hooks/useDropdown";
import type { SelectFieldOption } from "./SelectField";

export const useSelectField = (
  options: SelectFieldOption[],
  value: string,
  onChange: (value: string) => void
) => {
  const dropdown = useDropdown(options, value, onChange);
  const { open, setOpen, wrapperRef, listRef } = dropdown;
  const selected = options.find((option) => option.value === value);
  const disabled = options.length === 0;
  const isOpen = open && !disabled;

  React.useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled, setOpen]);

  useAnchoredList({
    isOpen,
    anchorRef: wrapperRef,
    listRef,
    shouldMatchWidth: true,
  });

  return { ...dropdown, selected, disabled, isOpen };
};
