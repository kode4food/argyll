import React, { useState, lazy, Suspense } from "react";
import { Activity, Play, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { generateFlowId } from "@/utils/flowUtils";
import { mapFlowStatusToProgressStatus } from "./FlowSelector/flowSelectorUtils";
import { useFlowDropdownManagement } from "./FlowSelector/useFlowDropdownManagement";
import { useFlowStatusUpdates } from "./FlowSelector/useFlowStatusUpdates";

const FlowCreateForm = lazy(() => import("./FlowCreateForm"));
const KeyboardShortcutsModal = lazy(
  () => import("../molecules/KeyboardShortcutsModal")
);

import { useFlowFromUrl } from "./FlowSelector/useFlowFromUrl";
import { useUI } from "../../contexts/UIContext";
import { getProgressIcon } from "@/utils/progressUtils";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import ErrorBoundary from "./ErrorBoundary";
import styles from "./FlowSelector.module.css";
import {
  FlowCreationStateProvider,
  useFlowCreation,
} from "../../contexts/FlowCreationContext";
import {
  FlowDropdownProvider,
  useFlowDropdownContext,
  FlowDropdownContextValue,
} from "../../contexts/FlowDropdownContext";
import { useFlowSession } from "../../contexts/FlowSessionContext";

const FlowSelectorDropdown = () => {
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
  } = useFlowDropdownContext();

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
          "Select Flow"
        )}
      </button>
      {showDropdown && (
        <div className={styles.dropdownMenu} ref={dropdownRef}>
          <div className={styles.dropdownSearch}>
            <Search className={styles.dropdownSearchIcon} />
            <input
              ref={searchInputRef}
              type="text"
              placeholder="Search flows..."
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
              No flows found
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const FlowSelectorContent: React.FC = () => {
  const router = useRouter();
  useFlowFromUrl();
  const { flows, selectedFlow, loadFlows, updateFlowStatus } = useFlowSession();
  const { subscribe, events } = useWebSocketContext();
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
        description: "Focus search",
        handler: () => {
          if (!showDropdown) {
            setShowDropdown(true);
            setTimeout(() => searchInputRef.current?.focus(), 100);
          }
        },
      },
      {
        key: "?",
        description: "Show keyboard shortcuts",
        handler: () => {
          setShowShortcutsModal(true);
        },
      },
    ],
    !showCreateForm && !showShortcutsModal
  );

  useFlowStatusUpdates({
    showDropdown,
    selectedFlow,
    subscribe,
    events,
    flows,
    updateFlowStatus,
    loadFlows,
  });

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
              aria-label="Argyll Web Site"
            >
              <Image
                src="/argyll-logo.png"
                alt="Argyll Logo"
                className={styles.icon}
                width={123}
                height={77}
              />
              <h1 className={styles.titleText}>Argyll Engine</h1>
            </a>
          </div>

          <div className={styles.right}>
            <div className={styles.controls}>
              <FlowSelectorDropdown />
              {selectedFlow ? (
                <button
                  onClick={() => router.push("/")}
                  className={styles.navButton}
                  title="Back to Overview"
                  aria-label="Back to Overview"
                >
                  <Activity className={styles.buttonIcon} aria-hidden="true" />
                </button>
              ) : (
                <>
                  <button
                    onClick={() => {
                      setNewID(generateFlowId());
                      setShowCreateForm(!showCreateForm);
                    }}
                    className={styles.createButton}
                    title="New Flow"
                    aria-label="Create New Flow"
                  >
                    <Play className={styles.buttonIcon} aria-hidden="true" />
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
        <ErrorBoundary
          title="Flow Form Error"
          description="An error occurred in the flow creation form. Try closing and reopening the form."
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
