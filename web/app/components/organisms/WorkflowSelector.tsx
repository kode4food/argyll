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
import { api, WorkflowStatus } from "../../api";
import toast from "react-hot-toast";
import { sortStepsByType } from "@/utils/stepUtils";
import {
  generateWorkflowId,
  generatePadded,
  sanitizeWorkflowID,
} from "@/utils/workflowUtils";

const WorkflowCreateForm = lazy(() => import("./WorkflowCreateForm"));
const KeyboardShortcutsModal = lazy(
  () => import("../molecules/KeyboardShortcutsModal")
);

import {
  useWorkflows,
  useSelectedWorkflow,
  useSteps,
  useLoadWorkflows,
  useAddWorkflow,
  useRemoveWorkflow,
  useUpdateWorkflowStatus,
} from "../../store/workflowStore";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { useWorkflowFromUrl } from "../../hooks/useWorkflowFromUrl";
import { useThrottledValue } from "../../hooks/useThrottledValue";
import { useUI } from "../../contexts/UIContext";
import { getProgressIcon, getProgressIconClass } from "@/utils/progressUtils";
import { StepProgressStatus } from "../../hooks/useStepProgress";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import ErrorBoundary from "./ErrorBoundary";
import styles from "./WorkflowSelector.module.css";

const mapWorkflowStatusToProgressStatus = (
  status: WorkflowStatus
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

const WorkflowSelector: React.FC = () => {
  const router = useRouter();
  useWorkflowFromUrl();
  const workflows = useWorkflows();
  const selectedWorkflow = useSelectedWorkflow();
  const steps = useSteps();
  const loadWorkflows = useLoadWorkflows();
  const addWorkflow = useAddWorkflow();
  const removeWorkflow = useRemoveWorkflow();
  const updateWorkflowStatus = useUpdateWorkflowStatus();
  const { subscribe, events } = useWebSocketContext();
  const {
    showCreateForm,
    setShowCreateForm,
    updatePreviewPlan,
    clearPreviewPlan,
    selectedStep,
    setSelectedStep,
    goalStepIds,
    setGoalStepIds,
  } = useUI();
  const [newId, setNewId] = useState("");
  const [initialState, setInitialState] = useState("{}");
  const [creating, setCreating] = useState(false);
  const [idManuallyEdited, setIDManuallyEdited] = useState(false);

  const throttled = useThrottledValue(initialState, 500);

  useEffect(() => {
    initialStateRef.current = initialState;
  }, [initialState]);
  const [showDropdown, setShowDropdown] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [showShortcutsModal, setShowShortcutsModal] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const prevStepRef = useRef<string | null>(null);
  const processedEventsRef = useRef<Set<string>>(new Set());
  const initialStateRef = useRef<string>(initialState);

  const handleCreateWorkflow = async () => {
    if (!newId.trim() || goalStepIds.length === 0) return;

    const workflowId = newId.trim();
    let parsedState: {};
    try {
      parsedState = JSON.parse(initialState);
    } catch {
      parsedState = {};
    }

    const optimisticWorkflow: any = {
      id: workflowId,
      status: "pending",
      goal_step_ids: goalStepIds,
      state: parsedState,
      started_at: new Date().toISOString(),
      execution_plan: null,
    };

    addWorkflow(optimisticWorkflow);
    setCreating(true);
    router.push(`/workflow/${workflowId}`);
    setNewId("");
    setGoalStepIds([]);
    setSelectedStep(null);
    setInitialState("{}");
    setShowCreateForm(false);

    try {
      await api.startWorkflow(workflowId, goalStepIds, parsedState);

      await loadWorkflows();
    } catch (error: any) {
      let errorMessage = "Unknown error";

      if (error?.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error?.message) {
        errorMessage = error.message;
      }

      removeWorkflow(workflowId);
      toast.error("Failed to create workflow: " + errorMessage);
      router.push("/");
    } finally {
      setCreating(false);
    }
  };

  const handleGoalStepChange = useCallback(
    async (stepIds: string[]) => {
      if (stepIds.length > 0) {
        try {
          let currentState: Record<string, any> = {};
          try {
            currentState = JSON.parse(initialStateRef.current);
          } catch {
            currentState = {};
          }

          const nonEmptyState: Record<string, any> = {};
          Object.keys(currentState).forEach((key) => {
            if (currentState[key] !== "") {
              nonEmptyState[key] = currentState[key];
            }
          });

          const executionPlan = await api.getExecutionPlan(
            stepIds,
            nonEmptyState
          );

          const mergedState: Record<string, any> = {};

          Object.keys(currentState).forEach((key) => {
            if (currentState[key] !== "") {
              mergedState[key] = currentState[key];
            }
          });

          executionPlan.required_inputs.forEach((name) => {
            if (!(name in mergedState)) {
              mergedState[name] = "";
            }
          });

          setInitialState(JSON.stringify(mergedState, null, 2));

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

          if (stepIds.length > 1) {
            const lastGoal = stepIds[stepIds.length - 1];
            const previousGoals = stepIds.slice(0, -1);

            try {
              const lastGoalPlan = await api.getExecutionPlan([lastGoal], {});
              const lastGoalStepIds = new Set(
                (lastGoalPlan.steps || []).map((s) => s.id)
              );

              const remainingGoals = previousGoals.filter(
                (id) => !lastGoalStepIds.has(id)
              );

              const finalGoals = [...remainingGoals, lastGoal];

              if (finalGoals.length !== stepIds.length) {
                setGoalStepIds(finalGoals);
                await updatePreviewPlan(finalGoals, mergedState);
                return;
              }
            } catch {}
          }

          setGoalStepIds(stepIds);
          await updatePreviewPlan(stepIds, mergedState);
        } catch (error) {
          clearPreviewPlan();
          setGoalStepIds(stepIds);
        }
      } else {
        let currentState: Record<string, any> = {};
        try {
          currentState = JSON.parse(initialStateRef.current);
        } catch {
          currentState = {};
        }

        const mergedState: Record<string, any> = {};
        Object.keys(currentState).forEach((key) => {
          if (currentState[key] !== "") {
            mergedState[key] = currentState[key];
          }
        });

        setInitialState(JSON.stringify(mergedState, null, 2));
        clearPreviewPlan();
        setGoalStepIds(stepIds);
      }
    },
    [
      idManuallyEdited,
      steps,
      setGoalStepIds,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  useEffect(() => {
    if (goalStepIds.length > 0 && showCreateForm) {
      let parsedState: Record<string, any>;
      try {
        parsedState = JSON.parse(throttled);
      } catch {
        parsedState = {};
      }
      updatePreviewPlan(goalStepIds, parsedState);
    } else if (!showCreateForm) {
      clearPreviewPlan();
    }
  }, [
    throttled,
    showCreateForm,
    updatePreviewPlan,
    clearPreviewPlan,
    goalStepIds,
  ]);

  const filteredWorkflows = workflows.filter((workflow) =>
    workflow.id.includes(sanitizeWorkflowID(searchTerm))
  );

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    setSelectedIndex(-1);
  };

  const selectableItems = filteredWorkflows.map((w) => w.id);

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
            router.push(`/workflow/${selectedItem}`);
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
            router.push(`/workflow/${selectedItem}`);
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
      prevStepRef.current = null;
    } else {
      router.prefetch("/workflow/placeholder");
    }
  }, [
    showCreateForm,
    clearPreviewPlan,
    setGoalStepIds,
    setSelectedStep,
    router,
  ]);

  useEffect(() => {
    if (!showCreateForm) return;
    if (prevStepRef.current === selectedStep) return;

    prevStepRef.current = selectedStep;
    const newGoalStepIds = selectedStep ? [selectedStep] : [];
    handleGoalStepChange(newGoalStepIds);
  }, [handleGoalStepChange, selectedStep, showCreateForm]);

  useEffect(() => {
    if (showDropdown || !selectedWorkflow) {
      subscribe({
        event_types: [
          "workflow_started",
          "workflow_completed",
          "workflow_failed",
        ],
      });
    } else {
      subscribe({
        event_types: [],
      });
    }
  }, [showDropdown, selectedWorkflow, subscribe]);

  useEffect(() => {
    const latestEvent = events[events.length - 1];
    if (!latestEvent) return;

    const aggregateId = latestEvent.aggregate_id;
    if (!aggregateId || aggregateId.length < 2) {
      return;
    }

    const eventKey = `${aggregateId.join(":")}:${latestEvent.sequence}`;

    if (processedEventsRef.current.has(eventKey)) {
      return;
    }

    processedEventsRef.current.add(eventKey);

    const eventType = latestEvent.type;
    const workflowId = aggregateId[1];

    if (eventType === "workflow_started") {
      const workflowExists = workflows.some((w) => w.id === workflowId);
      if (workflowExists) {
        updateWorkflowStatus(workflowId, "active");
      } else {
        loadWorkflows();
      }
    } else if (eventType === "workflow_completed") {
      updateWorkflowStatus(
        workflowId,
        "completed",
        new Date(latestEvent.timestamp).toISOString()
      );
    } else if (eventType === "workflow_failed") {
      updateWorkflowStatus(
        workflowId,
        "failed",
        new Date(latestEvent.timestamp).toISOString()
      );
    }
  }, [events, workflows, updateWorkflowStatus, loadWorkflows]);

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
                {selectedWorkflow ? (
                  <>
                    {(() => {
                      const workflow = workflows.find(
                        (w) => w.id === selectedWorkflow
                      );
                      const progressStatus = mapWorkflowStatusToProgressStatus(
                        workflow?.status || "pending"
                      );
                      const StatusIcon = getProgressIcon(progressStatus);
                      const iconClass = getProgressIconClass(progressStatus);
                      return (
                        <StatusIcon className={`progress-icon ${iconClass}`} />
                      );
                    })()}
                    {selectedWorkflow}
                  </>
                ) : (
                  "Select Workflow"
                )}
              </button>
              {showDropdown && (
                <div className={styles.dropdownMenu} ref={dropdownRef}>
                  <div className={styles.dropdownSearch}>
                    <Search className={styles.dropdownSearchIcon} />
                    <input
                      ref={searchInputRef}
                      type="text"
                      placeholder="Search workflows..."
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
                  {filteredWorkflows.map((workflow, index) => {
                    const progressStatus = mapWorkflowStatusToProgressStatus(
                      workflow.status
                    );
                    const StatusIcon = getProgressIcon(progressStatus);
                    const iconClass = getProgressIconClass(progressStatus);
                    return (
                      <div
                        key={workflow.id}
                        className={`${styles.dropdownItem} ${selectedIndex === index ? "bg-neutral-bg-dark" : ""} ${selectedWorkflow === workflow.id ? styles.dropdownItemSelected : ""}`}
                        onMouseDown={(e) => {
                          e.preventDefault();
                          router.push(`/workflow/${workflow.id}`);
                          setShowDropdown(false);
                          setSearchTerm("");
                          setSelectedIndex(-1);
                        }}
                      >
                        <StatusIcon className={`progress-icon ${iconClass}`} />
                        {workflow.id}
                      </div>
                    );
                  })}
                  {filteredWorkflows.length === 0 && searchTerm && (
                    <div
                      className={`${styles.dropdownItem} ${styles.noResults}`}
                    >
                      No workflows found
                    </div>
                  )}
                </div>
              )}
            </div>
            {selectedWorkflow ? (
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
                  onClick={async () => {
                    setNewId(generateWorkflowId());
                    if (selectedStep && !showCreateForm) {
                      await handleGoalStepChange([selectedStep]);
                    }
                    setShowCreateForm(!showCreateForm);
                  }}
                  className={styles.createButton}
                  title="New Workflow"
                  aria-label="Create New Workflow"
                >
                  <Play className="h-4 w-4" aria-hidden="true" />
                </button>
              </>
            )}
          </div>
        </div>
      </div>
      <ErrorBoundary
        title="Workflow Form Error"
        description="An error occurred in the workflow creation form. Try closing and reopening the form."
        onError={(error, errorInfo) => {
          console.error("WorkflowCreateForm error:", error, errorInfo);
          setShowCreateForm(false);
        }}
      >
        <Suspense fallback={null}>
          <WorkflowCreateForm
            newID={newId}
            setNewID={setNewId}
            setIDManuallyEdited={setIDManuallyEdited}
            handleStepChange={handleGoalStepChange}
            initialState={initialState}
            setInitialState={setInitialState}
            creating={creating}
            handleCreateWorkflow={handleCreateWorkflow}
            steps={steps}
            generateID={generateWorkflowId}
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

export default WorkflowSelector;
