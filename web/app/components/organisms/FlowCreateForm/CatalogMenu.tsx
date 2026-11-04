import React from "react";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import FeedManager from "@/app/components/organisms/FeedManager";
import SpaceManager from "@/app/components/organisms/SpaceManager";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import useDropdown from "@/app/hooks/useDropdown";
import { useT } from "@/app/i18n";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import { IconFeed, IconManage, IconSpace } from "@/utils/iconRegistry";
import styles from "./FlowCreateForm.module.css";

type CatalogManager = "spaces" | "feeds";

const CATALOG_ITEMS = [
  { value: "spaces", labelKey: "spaceManager.title", Icon: IconSpace },
  { value: "feeds", labelKey: "feedManager.title", Icon: IconFeed },
] as const;

const CatalogMenu: React.FC = () => {
  const t = useT();
  const [manager, setManager] = React.useState<CatalogManager | null>(null);
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown([...CATALOG_ITEMS], "", (value) =>
    setManager(value as CatalogManager)
  );

  const listRef = React.useRef<HTMLDivElement>(null);
  useAnchoredList({
    isOpen: open,
    anchorRef: wrapperRef,
    listRef,
    align: "end",
  });

  // The managers sit outside the wrapper: their dialogs render in place, so
  // keys typed in them would bubble into the menu's key handling
  return (
    <>
      <div
        ref={wrapperRef}
        className={styles.catalogMenu}
        onKeyDown={handleKeyDown}
      >
        <button
          type="button"
          className={styles.spaceManageButton}
          onClick={() => setOpen((isOpen) => !isOpen)}
          title={t("flowCreate.catalogMenu")}
          aria-label={t("flowCreate.catalogMenu")}
          aria-expanded={open}
          aria-haspopup="menu"
        >
          <IconManage className={styles.spaceManageIcon} />
        </button>
        {open && (
          <div
            ref={listRef}
            className={dropdownStyles.list}
            role="menu"
            data-ui-overlay="dropdown"
          >
            {CATALOG_ITEMS.map(({ value, labelKey, Icon }, index) => (
              <DropdownOption
                key={value}
                role="menuitem"
                icon={<Icon className={styles.spaceManageIcon} />}
                isHighlighted={index === highlightedIndex}
                onHover={() => setHighlightedIndex(index)}
                onSelect={() => {
                  setManager(value);
                  setOpen(false);
                }}
              >
                {t(labelKey)}
              </DropdownOption>
            ))}
          </div>
        )}
      </div>
      <SpaceManager
        isOpen={manager === "spaces"}
        onClose={() => setManager(null)}
      />
      {manager === "feeds" && (
        <FeedManager isOpen onClose={() => setManager(null)} />
      )}
    </>
  );
};

export default CatalogMenu;
