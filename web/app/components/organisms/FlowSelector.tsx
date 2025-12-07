import React, { useState, useRef, useEffect, lazy, Suspense } from "react";
import { Activity, Play, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { FlowStatus } from "../../api";
import { generateFlowId, sanitizeFlowID } from "@/utils/flowUtils";

const FlowCreateForm = lazy(() => import("./FlowCreateForm"));
const KeyboardShortcutsModal = lazy(
  () => import("../molecules/KeyboardShortcutsModal")
);

import {
  useFlows,
  useSelectedFlow,
  useLoadFlows,
  useUpdateFlowStatus,
} from "../../store/flowStore";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { useFlowFromUrl } from "../../hooks/useFlowFromUrl";
import { useUI } from "../../contexts/UIContext";
import { getProgressIcon } from "@/utils/progressUtils";
import { StepProgressStatus } from "../../hooks/useStepProgress";
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
} from "../../contexts/FlowDropdownContext";

const mapFlowStatusToProgressStatus = (
  status: FlowStatus
): StepProgressStatus => {
  switch (status) {
    case "pending":
      return "pending";
    case "active":
      return "active";
    case "completed":
      return "completed";
    case "failed":
      return "failed";
    default:
      return "pending";
  }
};

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
                flow?.status || "pending"
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
            const progressStatus = mapFlowStatusToProgressStatus(
              flow.status as FlowStatus
            );
            const StatusIcon = getProgressIcon(progressStatus);
            return (
              <div
                key={flow.id}
                className={`${styles.dropdownItem} ${selectedIndex === index ? "bg-neutral-bg-dark" : ""} ${selectedFlow === flow.id ? styles.dropdownItemSelected : ""}`}
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

const useFlowDropdown = (
  flows: ReturnType<typeof useFlows>,
  selectedFlow: string | null,
  router: ReturnType<typeof useRouter>
) => {
  const [showDropdown, setShowDropdown] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredFlows = flows.filter((flow) =>
    flow.id.includes(sanitizeFlowID(searchTerm))
  );
  const selectableItems = filteredFlows.map((w) => w.id);

  const closeDropdown = () => {
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    setSelectedIndex(-1);
  };

  const navigateToFlow = (flowId: string) => {
    if (flowId === "Overview") {
      router.push("/");
    } else {
      router.push(`/flow/${flowId}`);
    }
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!showDropdown) return;

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < selectableItems.length - 1 ? prev + 1 : 0
        );
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev > 0 ? prev - 1 : selectableItems.length - 1
        );
        break;
      case "Enter":
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < selectableItems.length) {
          navigateToFlow(selectableItems[selectedIndex]);
        }
        break;
      case "Tab":
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < selectableItems.length) {
          navigateToFlow(selectableItems[selectedIndex]);
        } else {
          setSelectedIndex(0);
        }
        break;
    }
  };

  useEffect(() => {
    if (selectedIndex >= 0 && dropdownRef.current) {
      const selectedElement = dropdownRef.current.children[
        selectedIndex + 1
      ] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({
          behavior: "smooth",
          block: "nearest",
        });
      }
    }
  }, [selectedIndex]);

  useEscapeKey(showDropdown, closeDropdown);

  return {
    showDropdown,
    setShowDropdown,
    searchTerm,
    selectedIndex,
    searchInputRef,
    dropdownRef,
    filteredFlows,
    handleSearchChange,
    handleKeyDown,
    selectFlow: navigateToFlow,
    closeDropdown,
    selectedFlow,
    flows,
  };
};

const useFlowEvents = ({
  showDropdown,
  selectedFlow,
  subscribe,
  events,
  flows,
  updateFlowStatus,
  loadFlows,
}: {
  showDropdown: boolean;
  selectedFlow: string | null;
  subscribe: ReturnType<typeof useWebSocketContext>["subscribe"];
  events: ReturnType<typeof useWebSocketContext>["events"];
  flows: ReturnType<typeof useFlows>;
  updateFlowStatus: ReturnType<typeof useUpdateFlowStatus>;
  loadFlows: ReturnType<typeof useLoadFlows>;
}) => {
  const processedEventsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (showDropdown || !selectedFlow) {
      subscribe({
        event_types: ["flow_started", "flow_completed", "flow_failed"],
      });
    } else {
      subscribe({
        event_types: [],
      });
    }
  }, [showDropdown, selectedFlow, subscribe]);

  useEffect(() => {
    const latestEvent = events[events.length - 1];
    if (!latestEvent) return;

    const id = latestEvent.id;
    if (!id || id.length < 2) {
      return;
    }

    const eventKey = `${id.join(":")}:${latestEvent.sequence}`;

    if (processedEventsRef.current.has(eventKey)) {
      return;
    }

    processedEventsRef.current.add(eventKey);

    const eventType = latestEvent.type;
    const flowId = id[1];

    if (eventType === "flow_started") {
      const flowExists = flows.some((w) => w.id === flowId);
      if (flowExists) {
        updateFlowStatus(flowId, "active");
      } else {
        loadFlows();
      }
    } else if (eventType === "flow_completed") {
      updateFlowStatus(
        flowId,
        "completed",
        new Date(latestEvent.timestamp).toISOString()
      );
    } else if (eventType === "flow_failed") {
      updateFlowStatus(
        flowId,
        "failed",
        new Date(latestEvent.timestamp).toISOString()
      );
    }
  }, [events, flows, updateFlowStatus, loadFlows]);
};

const FlowSelectorContent: React.FC = () => {
  const router = useRouter();
  useFlowFromUrl();
  const flows = useFlows();
  const selectedFlow = useSelectedFlow();
  const loadFlows = useLoadFlows();
  const updateFlowStatus = useUpdateFlowStatus();
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
  } = useFlowDropdown(flows, selectedFlow, router);

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

  useFlowEvents({
    showDropdown,
    selectedFlow,
    subscribe,
    events,
    flows,
    updateFlowStatus,
    loadFlows,
  });

  const dropdownValue: Parameters<typeof FlowDropdownProvider>[0]["value"] = {
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
            <div className={styles.title}>
              <Image
                src="/spuds-logo.svg"
                alt="Spuds Logo"
                className={styles.icon}
                width={123}
                height={77}
              />
              <h1 className={styles.titleText}>Spuds Engine</h1>
            </div>
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
                  <Activity className="h-4 w-4" aria-hidden="true" />
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
                    <Play className="h-4 w-4" aria-hidden="true" />
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
