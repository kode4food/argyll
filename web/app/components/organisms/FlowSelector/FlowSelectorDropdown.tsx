import React from "react";
import { createPortal } from "react-dom";
import { IconSearch } from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import { useFlowDropdownManagement } from "./useFlowDropdownManagement";
import { useFlowSelectorDropdown } from "./useFlowSelectorDropdown";
import styles from "./FlowSelector.module.css";

const BLUR_CLOSE_DELAY_MS = 100;

type FlowSelectorDropdownProps = ReturnType<
  typeof useFlowDropdownManagement
> & {
  flowsHasMore: boolean;
  flowsLoading: boolean;
  loadMoreFlows: () => Promise<void>;
};

const FlowSelectorDropdown: React.FC<FlowSelectorDropdownProps> = ({
  showDropdown,
  setShowDropdown,
  searchTerm,
  selectedIndex,
  setSelectedIndex,
  searchInputRef,
  dropdownRef,
  filteredFlows,
  handleSearchChange,
  handleKeyDown,
  setScrollTop,
  selectedFlow,
  selectFlow,
  flows,
  closeDropdown,
  flowsHasMore,
  flowsLoading,
  loadMoreFlows,
}) => {
  const t = useT();
  const {
    handleDropdownScroll,
    triggerRef,
    menuRef,
    flowRows,
    selectedProgressStatus,
    SelectedStatusIcon,
  } = useFlowSelectorDropdown({
    showDropdown,
    selectedIndex,
    selectedFlow,
    filteredFlows,
    flows,
    setScrollTop,
    flowsHasMore,
    flowsLoading,
    loadMoreFlows,
  });

  const dropdownMenu = showDropdown ? (
    <div
      className={styles.dropdownMenuPortal}
      ref={menuRef}
      data-ui-overlay="dropdown"
    >
      <div
        className={styles.dropdownMenu}
        ref={dropdownRef}
        onScroll={handleDropdownScroll}
      >
        <div className={styles.dropdownSearch}>
          <IconSearch className={styles.dropdownSearchIcon} />
          <input
            ref={searchInputRef}
            type="text"
            placeholder={t("flowSelector.searchPlaceholder")}
            value={searchTerm}
            onChange={handleSearchChange}
            onKeyDown={handleKeyDown}
            onBlur={() =>
              setTimeout(() => closeDropdown(), BLUR_CLOSE_DELAY_MS)
            }
            className={styles.dropdownSearchInput}
            autoFocus
          />
        </div>
        {flowRows.map(
          ({ flow, index, className, progressStatus, StatusIcon }) => (
            <button
              type="button"
              key={flow.id}
              data-flow-id={flow.id}
              className={className}
              onMouseEnter={() => setSelectedIndex(index)}
              onMouseDown={(e) => {
                e.preventDefault();
                selectFlow(flow.id);
                closeDropdown();
              }}
            >
              <StatusIcon
                className={`progress-icon ${progressStatus || "pending"}`}
              />
              {flow.id}
            </button>
          )
        )}
        {filteredFlows.length === 0 && searchTerm && (
          <div
            className={`${dropdownStyles.item} ${styles.dropdownItem} ${styles.noResults}`}
          >
            {t("flowSelector.noFlowsFound")}
          </div>
        )}
        {flowsLoading && (
          <div
            className={`${dropdownStyles.item} ${styles.dropdownItem} ${styles.noResults}`}
          >
            {t("flowSelector.loadingMore")}
          </div>
        )}
        {!flowsLoading && flowsHasMore && (
          <button
            type="button"
            className={`${dropdownStyles.item} ${styles.dropdownItem} ${styles.loadMore}`}
            onMouseDown={(e) => {
              e.preventDefault();
              void loadMoreFlows();
            }}
          >
            {t("flowSelector.loadMore")}
          </button>
        )}
      </div>
    </div>
  ) : null;

  return (
    <div className={styles.dropdown} ref={triggerRef}>
      <button
        onClick={() => setShowDropdown(!showDropdown)}
        className={styles.select}
      >
        {selectedFlow ? (
          <>
            <SelectedStatusIcon
              className={`progress-icon ${selectedProgressStatus || "pending"}`}
            />
            {selectedFlow}
          </>
        ) : (
          t("flowSelector.selectFlow")
        )}
      </button>
      {dropdownMenu && createPortal(dropdownMenu, document.body)}
    </div>
  );
};

export default FlowSelectorDropdown;
