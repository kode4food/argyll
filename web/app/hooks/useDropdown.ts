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
  listRef: React.RefObject<HTMLDivElement | null>;
  handleKeyDown: (e: React.KeyboardEvent) => void;
  selectValue: (value: string) => void;
}

const OPEN_KEYS = new Set(["ArrowDown", "ArrowUp", " "]);
const SELECT_KEYS = new Set(["Enter", " "]);
const CLOSE_KEYS = new Set(["Escape", "Tab"]);
const NAV_DIRECTIONS: Record<string, 1 | -1> = { ArrowDown: 1, ArrowUp: -1 };

const findEnabledIndex = (
  options: DropdownItem[],
  current: number,
  direction: 1 | -1
): number => {
  let next = current + direction;
  while (next >= 0 && next < options.length) {
    if (!options[next].disabled) return next;
    next += direction;
  }
  return current;
};

const scrollSelectedIntoView = (list: Element | null | undefined) => {
  if (!list || list.scrollHeight <= list.clientHeight) return;
  list.querySelector('[aria-selected="true"]')?.scrollIntoView({
    block: "nearest",
  });
};

const useDropdown = <T extends DropdownItem>(
  options: T[],
  value: string,
  onSelect: (value: string) => void
): UseDropdownResult => {
  const [open, setOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      const idx = options.findIndex((o) => o.value === value);
      setHighlightedIndex(idx >= 0 ? idx : 0);
      scrollSelectedIntoView(
        listRef.current ?? wrapperRef.current?.querySelector('[role="listbox"]')
      );
    } else {
      setHighlightedIndex(-1);
    }
    // intentionally omitting options/value: only reset on open/close transition
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useClickOutside(
    wrapperRef,
    (event) => {
      if (!listRef.current?.contains(event.target as Node)) setOpen(false);
    },
    open
  );

  const selectValue = (selected: string) => {
    onSelect(selected);
    setOpen(false);
  };

  const selectHighlighted = () => {
    const opt = options[highlightedIndex];
    if (opt && !opt.disabled) {
      selectValue(opt.value);
    }
  };

  const handleOpenKeyDown = (e: React.KeyboardEvent) => {
    const direction = NAV_DIRECTIONS[e.key];
    if (direction) {
      e.preventDefault();
      setHighlightedIndex((current) =>
        findEnabledIndex(options, current, direction)
      );
    } else if (SELECT_KEYS.has(e.key)) {
      e.preventDefault();
      selectHighlighted();
    } else if (CLOSE_KEYS.has(e.key)) {
      if (e.key === "Escape") e.preventDefault();
      setOpen(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (open) {
      handleOpenKeyDown(e);
      return;
    }
    if (OPEN_KEYS.has(e.key)) {
      e.preventDefault();
      setOpen(true);
    }
  };

  return {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    listRef,
    handleKeyDown,
    selectValue,
  };
};

export default useDropdown;
