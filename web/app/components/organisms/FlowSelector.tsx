import React, {
  useState,
  useRef,
  useEffect,
  useCallback,
  lazy,
  Suspense,
} from "react";
import { Activity, Play, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { api, FlowStatus } from "../../api";
import toast from "react-hot-toast";
import { sortStepsByType } from "@/utils/stepUtils";
import {
  generateFlowId,
  generatePadded,
  sanitizeFlowID,
} from "@/utils/flowUtils";
import {
  parseState,
  filterDefaultValues,
  addRequiredDefaults,
} from "@/utils/stateUtils";

const FlowCreateForm = lazy(() => import("./FlowCreateForm"));
const KeyboardShortcutsModal = lazy(
  () => import("../molecules/KeyboardShortcutsModal")
);

import {
  useFlows,
  useSelectedFlow,
  useSteps,
  useLoadFlows,
  useAddFlow,
  useRemoveFlow,
  useUpdateFlowStatus,
} from "../../store/flowStore";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { useFlowFromUrl } from "../../hooks/useFlowFromUrl";
import { useUI } from "../../contexts/UIContext";
import { getProgressIcon, getProgressIconClass } from "@/utils/progressUtils";
import { StepProgressStatus } from "../../hooks/useStepProgress";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import ErrorBoundary from "./ErrorBoundary";
import styles from "./FlowSelector.module.css";

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

const FlowSelector: React.FC = () => {
  const router = useRouter();
  useFlowFromUrl();
  const flows = useFlows();
  const selectedFlow = useSelectedFlow();
  const steps = useSteps();
  const loadFlows = useLoadFlows();
  const addFlow = useAddFlow();
  const removeFlow = useRemoveFlow();
  const updateFlowStatus = useUpdateFlowStatus();
  const { subscribe, events } = useWebSocketContext();
  const {
    showCreateForm,
    setShowCreateForm,
    previewPlan,
    updatePreviewPlan,
    clearPreviewPlan,
    setSelectedStep,
    goalStepIds,
    setGoalStepIds,
  } = useUI();
  const [newId, setNewId] = useState("");
  const [initialState, setInitialState] = useState("{}");
  const [creating, setCreating] = useState(false);
  const [idManuallyEdited, setIDManuallyEdited] = useState(false);

  const [showDropdown, setShowDropdown] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [showShortcutsModal, setShowShortcutsModal] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const processedEventsRef = useRef<Set<string>>(new Set());

  const handleCreateFlow = async () => {
    if (!newId.trim() || goalStepIds.length === 0) return;

    const flowId = newId.trim();
    let parsedState: {};
    try {
      parsedState = JSON.parse(initialState);
    } catch {
      parsedState = {};
    }

    addFlow({
      id: flowId,
      status: "pending",
      state: parsedState,
      started_at: new Date().toISOString(),
      plan: previewPlan || undefined,
    });

    setCreating(true);
    router.push(`/flow/${flowId}`);
    setNewId("");
    setGoalStepIds([]);
    setSelectedStep(null);
    setInitialState("{}");
    setShowCreateForm(false);

    try {
      await api.startFlow(flowId, goalStepIds, parsedState);

      await loadFlows();
    } catch (error: any) {
      let errorMessage = "Unknown error";

      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error?.message) {
        errorMessage = error.message;
      }

      removeFlow(flowId);
      toast.error("Failed to create flow: " + errorMessage);
      router.push("/");
    } finally {
      setCreating(false);
    }
  };

  const handleGoalStepChange = useCallback(
    async (stepIds: string[]) => {
      const currentState = parseState(initialState);
      const nonDefaultState = filterDefaultValues(currentState, steps);

      if (stepIds.length === 0) {
        setInitialState(JSON.stringify(nonDefaultState, null, 2));
        clearPreviewPlan();
        setGoalStepIds([]);
        return;
      }

      try {
        const executionPlan = await api.getExecutionPlan(
          stepIds,
          nonDefaultState
        );

        const stateWithDefaults = addRequiredDefaults(
          nonDefaultState,
          executionPlan
        );

        setInitialState(JSON.stringify(stateWithDefaults, null, 2));

        if (!idManuallyEdited) {
          const lastGoalId = stepIds[stepIds.length - 1];
          const goalStep = steps.find((s) => s.id === lastGoalId);
          const goalName = goalStep?.name || lastGoalId;
          const kebabName = goalName
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-+|-+$/g, "");
          setNewId(`${kebabName}-${generatePadded()}`);
        }

        // Handle multiple goals - remove redundant ones
        if (stepIds.length > 1) {
          const lastGoal = stepIds[stepIds.length - 1];
          const previousGoals = stepIds.slice(0, -1);

          try {
            const lastGoalPlan = await api.getExecutionPlan([lastGoal], {});
            const lastGoalStepIds = new Set(
              Object.keys(lastGoalPlan.steps || {})
            );

            const remainingGoals = previousGoals.filter(
              (id) => !lastGoalStepIds.has(id)
            );

            const finalGoals = [...remainingGoals, lastGoal];

            if (finalGoals.length !== stepIds.length) {
              setGoalStepIds(finalGoals);
              await updatePreviewPlan(finalGoals, stateWithDefaults);
              return;
            }
          } catch {}
        }

        setGoalStepIds(stepIds);
        await updatePreviewPlan(stepIds, stateWithDefaults);
      } catch (error) {
        clearPreviewPlan();
        setGoalStepIds(stepIds);
      }
    },
    [
      initialState,
      idManuallyEdited,
      steps,
      setGoalStepIds,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  const filteredFlows = flows.filter((flow) =>
    flow.id.includes(sanitizeFlowID(searchTerm))
  );

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    setSelectedIndex(-1);
  };

  const selectableItems = filteredFlows.map((w) => w.id);

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
          const selectedItem = selectableItems[selectedIndex];
          if (selectedItem === "Overview") {
            router.push("/");
          } else {
            router.push(`/flow/${selectedItem}`);
          }
          setShowDropdown(false);
          setSearchTerm("");
          setSelectedIndex(-1);
        }
        break;
      case "Tab":
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < selectableItems.length) {
          const selectedItem = selectableItems[selectedIndex];
          if (selectedItem === "Overview") {
            router.push("/");
          } else {
            router.push(`/flow/${selectedItem}`);
          }
          setShowDropdown(false);
          setSearchTerm("");
          setSelectedIndex(-1);
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

  useEscapeKey(showDropdown, () => {
    setShowDropdown(false);
    setSearchTerm("");
    setSelectedIndex(-1);
  });

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

  useEffect(() => {
    if (!showCreateForm) {
      setNewId("");
      setGoalStepIds([]);
      setSelectedStep(null);
      setInitialState("{}");
      setIDManuallyEdited(false);
      clearPreviewPlan();
    } else {
      router.prefetch("/flow/placeholder");
      // If there are already goal steps selected, initialize the form
      if (goalStepIds.length > 0) {
        handleGoalStepChange(goalStepIds);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showCreateForm]);

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

  return (
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
                      const iconClass = getProgressIconClass(progressStatus);
                      return (
                        <StatusIcon className={`progress-icon ${iconClass}`} />
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
                      onBlur={() =>
                        setTimeout(() => setShowDropdown(false), 100)
                      }
                      className={styles.dropdownSearchInput}
                      autoFocus
                    />
                  </div>
                  {filteredFlows.map((flow, index) => {
                    const progressStatus = mapFlowStatusToProgressStatus(
                      flow.status
                    );
                    const StatusIcon = getProgressIcon(progressStatus);
                    const iconClass = getProgressIconClass(progressStatus);
                    return (
                      <div
                        key={flow.id}
                        className={`${styles.dropdownItem} ${selectedIndex === index ? "bg-neutral-bg-dark" : ""} ${selectedFlow === flow.id ? styles.dropdownItemSelected : ""}`}
                        onMouseDown={(e) => {
                          e.preventDefault();
                          router.push(`/flow/${flow.id}`);
                          setShowDropdown(false);
                          setSearchTerm("");
                          setSelectedIndex(-1);
                        }}
                      >
                        <StatusIcon className={`progress-icon ${iconClass}`} />
                        {flow.id}
                      </div>
                    );
                  })}
                  {filteredFlows.length === 0 && searchTerm && (
                    <div
                      className={`${styles.dropdownItem} ${styles.noResults}`}
                    >
                      No flows found
                    </div>
                  )}
                </div>
              )}
            </div>
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
                    setNewId(generateFlowId());
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
          <FlowCreateForm
            newID={newId}
            setNewID={setNewId}
            setIDManuallyEdited={setIDManuallyEdited}
            handleStepChange={handleGoalStepChange}
            initialState={initialState}
            setInitialState={setInitialState}
            creating={creating}
            handleCreateFlow={handleCreateFlow}
            steps={steps}
            generateID={generateFlowId}
            sortSteps={sortStepsByType}
          />
        </Suspense>
      </ErrorBoundary>
      <Suspense fallback={null}>
        <KeyboardShortcutsModal
          isOpen={showShortcutsModal}
          onClose={() => setShowShortcutsModal(false)}
        />
      </Suspense>
    </div>
  );
};

export default FlowSelector;
