import React from "react";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import useDropdown from "@/app/hooks/useDropdown";
import { IconFeed, IconSpace } from "@/utils/iconRegistry";

export type CatalogManager = "spaces" | "feeds";

export const CATALOG_ITEMS = [
  { value: "spaces", labelKey: "spaceManager.title", Icon: IconSpace },
  { value: "feeds", labelKey: "feedManager.title", Icon: IconFeed },
] as const;

export const useCatalogMenu = () => {
  const [manager, setManager] = React.useState<CatalogManager | null>(null);
  const dropdown = useDropdown([...CATALOG_ITEMS], "", (value) =>
    setManager(value as CatalogManager)
  );
  const { open, setOpen, wrapperRef, selectValue } = dropdown;

  const listRef = React.useRef<HTMLDivElement>(null);
  useAnchoredList({
    isOpen: open,
    anchorRef: wrapperRef,
    listRef,
    align: "end",
  });

  const closeManager = () => setManager(null);
  const toggleOpen = () => setOpen((isOpen) => !isOpen);

  return {
    ...dropdown,
    listRef,
    manager,
    selectManager: selectValue,
    closeManager,
    toggleOpen,
  };
};
