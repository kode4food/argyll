import React from "react";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import { useChatAction } from "@/app/hooks/useChatAction";
import useDropdown from "@/app/hooks/useDropdown";
import { IconFeed, IconSpace } from "@/utils/iconRegistry";

export type CatalogManager = "spaces" | "feeds";

interface CatalogTarget {
  manager: CatalogManager;
  id?: string;
}

export const CATALOG_ITEMS = [
  { value: "spaces", labelKey: "spaceManager.title", Icon: IconSpace },
  { value: "feeds", labelKey: "feedManager.title", Icon: IconFeed },
] as const;

export const useCatalogMenu = () => {
  const [target, setTarget] = React.useState<CatalogTarget | null>(null);
  const [serial, setSerial] = React.useState(0);
  const openManager = (next: CatalogTarget) => {
    setTarget(next);
    setSerial((current) => current + 1);
  };
  const dropdown = useDropdown([...CATALOG_ITEMS], "", (value) =>
    openManager({ manager: value as CatalogManager })
  );

  useChatAction({
    open_space: ({ space_id }) =>
      openManager({ manager: "spaces", id: space_id }),
    open_feed: ({ feed_id }) => openManager({ manager: "feeds", id: feed_id }),
  });
  const { open, setOpen, wrapperRef, selectValue } = dropdown;

  const listRef = React.useRef<HTMLDivElement>(null);
  useAnchoredList({
    isOpen: open,
    anchorRef: wrapperRef,
    listRef,
    align: "end",
  });

  const closeManager = () => setTarget(null);
  const toggleOpen = () => setOpen((isOpen) => !isOpen);

  return {
    ...dropdown,
    listRef,
    target,
    serial,
    selectManager: selectValue,
    closeManager,
    toggleOpen,
  };
};
