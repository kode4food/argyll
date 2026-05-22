import { useState, useEffect, useRef } from "react";
import useClickOutside from "./useClickOutside";

interface DropdownItem {
  value: string;
  disabled?: boolean;
}

interface UseDropdownResult {
  open: boolean;
  setOpen: React.Dispatch<React.SetStateAction<boolean>>;
  highlightedIndex: number;
  setHighlightedIndex: React.Dispatch<React.SetStateAction<number>>;
  wrapperRef: React.RefObject<HTMLDivElement | null>;
  handleKeyDown: (e: React.KeyboardEvent) => void;
}

const useDropdown = <T extends DropdownItem>(
  options: T[],
  value: string,
  onSelect: (value: string) => void
): UseDropdownResult => {
  const [open, setOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const wrapperRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      const idx = options.findIndex((o) => o.value === value);
      setHighlightedIndex(idx >= 0 ? idx : 0);
    } else {
      setHighlightedIndex(-1);
    }
    // intentionally omitting options/value — only reset on open/close transition
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useClickOutside(wrapperRef, () => setOpen(false), open);

  const navigate = (direction: 1 | -1) => {
    setHighlightedIndex((current) => {
      let next = current + direction;
      while (next >= 0 && next < options.length) {
        if (!options[next].disabled) return next;
        next += direction;
      }
      return current;
    });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === " ") {
        e.preventDefault();
        setOpen(true);
      }
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      navigate(1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      navigate(-1);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      const opt = options[highlightedIndex];
      if (opt && !opt.disabled) {
        onSelect(opt.value);
        setOpen(false);
      }
    } else if (e.key === "Escape" || e.key === "Tab") {
      setOpen(false);
    }
  };

  return {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  };
};

export default useDropdown;
