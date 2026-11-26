import React from "react";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import FeedManager from "@/app/components/organisms/FeedManager";
import SpaceManager from "@/app/components/organisms/SpaceManager";
import { useT } from "@/app/i18n";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import { IconManage } from "@/utils/iconRegistry";
import styles from "./FlowCreateForm.module.css";
import { CATALOG_ITEMS, useCatalogMenu } from "./useCatalogMenu";

const CatalogMenu: React.FC = () => {
  const t = useT();
  const {
    open,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
    listRef,
    target,
    serial,
    selectManager,
    closeManager,
    toggleOpen,
  } = useCatalogMenu();

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
          onClick={toggleOpen}
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
                onSelect={() => selectManager(value)}
              >
                {t(labelKey)}
              </DropdownOption>
            ))}
          </div>
        )}
      </div>
      {target?.manager === "spaces" && (
        <SpaceManager
          key={serial}
          isOpen
          onClose={closeManager}
          spaceId={target.id}
          seed={target.seed}
        />
      )}
      {target?.manager === "feeds" && (
        <FeedManager
          key={serial}
          isOpen
          onClose={closeManager}
          feedId={target.id}
          seed={target.seed}
        />
      )}
    </>
  );
};

export default CatalogMenu;
