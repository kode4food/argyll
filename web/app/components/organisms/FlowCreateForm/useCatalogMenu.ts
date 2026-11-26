import React from "react";
import { Feed, Space } from "@/app/api";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import { useChatAction } from "@/app/hooks/useChatAction";
import useDropdown from "@/app/hooks/useDropdown";
import { IconFeed, IconSpace } from "@/utils/iconRegistry";

export type CatalogManager = "spaces" | "feeds";

type CatalogTarget =
  | { manager: "spaces"; id?: string; seed?: Partial<Space> }
  | { manager: "feeds"; id?: string; seed?: Partial<Feed> };

export const CATALOG_ITEMS = [
  { value: "spaces", labelKey: "spaceManager.title", Icon: IconSpace },
  { value: "feeds", labelKey: "feedManager.title", Icon: IconFeed },
] as const;

export const useCatalogMenu = () => {
  const [target, setTarget] = React.useState<CatalogTarget | null>(null);
  // A new key remounts the manager, so its initial state takes the new target
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
    new_space: ({ draft }) =>
      openManager({ manager: "spaces", seed: draft ?? {} }),
    open_feed: ({ feed_id }) => openManager({ manager: "feeds", id: feed_id }),
    new_feed: ({ draft }) =>
      openManager({ manager: "feeds", seed: draft ?? {} }),
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
