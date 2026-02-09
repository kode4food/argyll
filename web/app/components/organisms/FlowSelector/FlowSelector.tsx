import React, { useState, lazy, Suspense } from "react";
import {
  IconCreateFlow,
  IconNavigateOverview,
  IconSearch,
} from "@/utils/iconRegistry";
import { useNavigate } from "react-router-dom";
import { generateFlowId } from "@/utils/flowUtils";
import { mapFlowStatusToProgressStatus } from "./flowSelectorUtils";
import { useFlowDropdownManagement } from "./useFlowDropdownManagement";
import { useT } from "@/app/i18n";

const FlowCreateForm = lazy(() => import("../FlowCreateForm/FlowCreateForm"));
const KeyboardShortcutsModal = lazy(
  () => import("@/app/components/molecules/KeyboardShortcutsModal")
);

import { useFlowFromUrl } from "./useFlowFromUrl";
import { useUI } from "@/app/contexts/UIContext";
import { getProgressIcon } from "@/utils/progressUtils";
import { useKeyboardShortcuts } from "@/app/hooks/useKeyboardShortcuts";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import styles from "./FlowSelector.module.css";
import {
  FlowCreationStateProvider,
  useFlowCreation,
} from "@/app/contexts/FlowCreationContext";
import {
  FlowDropdownProvider,
  useFlowDropdownContext,
  FlowDropdownContextValue,
} from "@/app/contexts/FlowDropdownContext";
import { useFlowSession } from "@/app/contexts/FlowSessionContext";

const FlowSelectorDropdown = () => {
  const t = useT();
  const {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    selectedFlow,
    selectFlow,
    flows,
    closeDropdown,
    flowsHasMore,
    flowsLoading,
    loadMoreFlows,
  } = useFlowDropdownContext();

  const handleDropdownScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (!flowsHasMore || flowsLoading) {
      return;
    }
    const target = e.currentTarget;
    const nearBottom =
      target.scrollTop + target.clientHeight >= target.scrollHeight - 40;
    if (nearBottom) {
      void loadMoreFlows();
    }
  };

  return (
    <div className={styles.dropdown}>
      <button
        onClick={() => setShowDropdown(!showDropdown)}
        className={styles.select}
      >
        {selectedFlow ? (
          <>
            {(() => {
              const flow = flows.find((w) => w.id === selectedFlow);
              const progressStatus = mapFlowStatusToProgressStatus(
                flow?.status ?? "pending"
              );
              const StatusIcon = getProgressIcon(progressStatus);
              return (
                <StatusIcon
                  className={`progress-icon ${progressStatus || "pending"}`}
                />
              );
            })()}
            {selectedFlow}
          </>
        ) : (
          t("flowSelector.selectFlow")
        )}
      </button>
      {showDropdown && (
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
              onBlur={() => setTimeout(() => closeDropdown(), 100)}
              className={styles.dropdownSearchInput}
              autoFocus
            />
          </div>
          {filteredFlows.map((flow, index) => {
            const progressStatus = mapFlowStatusToProgressStatus(flow.status);
            const StatusIcon = getProgressIcon(progressStatus);
            const isHighlighted = selectedIndex === index;
            const isSelected = selectedFlow === flow.id;
            const dropdownItemClassName = [
              styles.dropdownItem,
              isHighlighted && styles.dropdownItemHighlighted,
              isSelected && styles.dropdownItemSelected,
            ]
              .filter(Boolean)
              .join(" ");
            return (
              <div
                key={flow.id}
                className={dropdownItemClassName}
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
              </div>
            );
          })}
          {filteredFlows.length === 0 && searchTerm && (
            <div className={`${styles.dropdownItem} ${styles.noResults}`}>
              {t("flowSelector.noFlowsFound")}
            </div>
          )}
          {flowsLoading && (
            <div className={`${styles.dropdownItem} ${styles.noResults}`}>
              {t("flowSelector.loadingMore")}
            </div>
          )}
          {!flowsLoading && flowsHasMore && (
            <div
              className={`${styles.dropdownItem} ${styles.loadMore}`}
              onMouseDown={(e) => {
                e.preventDefault();
                void loadMoreFlows();
              }}
            >
              {t("flowSelector.loadMore")}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const FlowSelectorContent: React.FC = () => {
  const t = useT();
  const navigate = useNavigate();
  useFlowFromUrl();
  const { flows, selectedFlow, flowsHasMore, flowsLoading, loadMoreFlows } =
    useFlowSession();
  const { showCreateForm, setShowCreateForm } = useUI();
  const { setNewID } = useFlowCreation();

  const {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    selectFlow,
    closeDropdown,
    selectedFlow: dropdownSelectedFlow,
    flows: dropdownFlows,
  } = useFlowDropdownManagement(flows, selectedFlow);

  const [showShortcutsModal, setShowShortcutsModal] = useState(false);

  useKeyboardShortcuts(
    [
      {
        key: "/",
        description: t("flowSelector.focusSearch"),
        handler: () => {
          if (!showDropdown) {
            setShowDropdown(true);
            setTimeout(() => searchInputRef.current?.focus(), 100);
          }
        },
      },
      {
        key: "?",
        description: t("flowSelector.showShortcuts"),
        handler: () => {
          setShowShortcutsModal(true);
        },
      },
    ],
    !showCreateForm && !showShortcutsModal
  );

  const dropdownValue: FlowDropdownContextValue = {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    selectFlow,
    closeDropdown,
    selectedFlow: dropdownSelectedFlow,
    flows: dropdownFlows,
    flowsHasMore,
    flowsLoading,
    loadMoreFlows,
  };

  return (
    <FlowDropdownProvider value={dropdownValue}>
      <div className={styles.selector}>
        <div className={styles.header}>
          <div className={styles.left}>
            <a
              href="https://www.argyll.app/"
              target="_blank"
              rel="noreferrer"
              className={`${styles.title} ${styles.titleLink}`}
              aria-label={t("flowSelector.siteLabel")}
            >
              <img
                src="/argyll-logo.png"
                alt={t("flowSelector.logoAlt")}
                className={styles.icon}
                width={123}
                height={77}
              />
              <h1 className={styles.titleText}>{t("flowSelector.title")}</h1>
            </a>
          </div>

          <div className={styles.right}>
            <div className={styles.controls}>
              <FlowSelectorDropdown />
              {selectedFlow ? (
                <button
                  onClick={() => navigate("/")}
                  className={styles.navButton}
                  title={t("flowSelector.backToOverview")}
                  aria-label={t("flowSelector.backToOverview")}
                >
                  <IconNavigateOverview
                    className={styles.buttonIcon}
                    aria-hidden="true"
                  />
                </button>
              ) : (
                <>
                  <button
                    onClick={() => {
                      setNewID(generateFlowId());
                      setShowCreateForm(!showCreateForm);
                    }}
                    className={styles.createButton}
                    title={t("flowSelector.newFlow")}
                    aria-label={t("flowSelector.createNewFlow")}
                  >
                    <IconCreateFlow
                      className={styles.buttonIcon}
                      aria-hidden="true"
                    />
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
        <ErrorBoundary
          title={t("flowSelector.formErrorTitle")}
          description={t("flowSelector.formErrorDescription")}
          onError={(error, errorInfo) => {
            console.error("Error in FlowCreateForm:", error);
            console.error("Component stack:", errorInfo.componentStack);
            setShowCreateForm(false);
          }}
        >
          <Suspense fallback={null}>
            <FlowCreateForm />
          </Suspense>
        </ErrorBoundary>
        <Suspense fallback={null}>
          <KeyboardShortcutsModal
            isOpen={showShortcutsModal}
            onClose={() => setShowShortcutsModal(false)}
          />
        </Suspense>
      </div>
    </FlowDropdownProvider>
  );
};

const FlowSelector: React.FC = () => (
  <FlowCreationStateProvider>
    <FlowSelectorContent />
  </FlowCreationStateProvider>
);

export default FlowSelector;
