import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";
import { api, ExecutionPlan } from "../api";

interface UIContextType {
  showCreateForm: boolean;
  setShowCreateForm: (show: boolean) => void;
  disableEdit: boolean;
  diagramContainerRef: React.RefObject<HTMLDivElement | null>;
  previewPlan: ExecutionPlan | null;
  goalSteps: string[];
  toggleGoalStep: (stepId: string) => void;
  setGoalSteps: (stepIds: string[]) => void;
  updatePreviewPlan: (
    goalSteps: string[],
    initialState: Record<string, any>
  ) => Promise<void>;
  clearPreviewPlan: () => void;
}

const UIContext = createContext<UIContextType | undefined>(undefined);

export const UIProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [previewPlan, setPreviewPlan] = useState<ExecutionPlan | null>(null);
  const [goalSteps, setGoalStepsState] = useState<string[]>([]);
  const diagramContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const disableEdit = showCreateForm;

  const setGoalSteps = useCallback((stepIds: string[]) => {
    setGoalStepsState(stepIds);
  }, []);

  const toggleGoalStep = useCallback((stepId: string) => {
    setGoalStepsState((prev) => {
      if (prev.includes(stepId)) {
        return prev.filter((id) => id !== stepId);
      }

      return [...prev, stepId];
    });
  }, []);

  const updatePreviewPlan = useCallback(
    async (goalSteps: string[], initialState: Record<string, any>) => {
      // Cancel any pending request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      if (goalSteps.length === 0) {
        setPreviewPlan(null);
        return;
      }

      // Create new abort controller for this request
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      try {
        const plan = await api.getExecutionPlan(
          goalSteps,
          initialState,
          abortController.signal
        );

        // Only update state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setPreviewPlan(plan);
        }
      } catch (error: any) {
        // Ignore abort errors
        if (error?.name !== "AbortError" && error?.code !== "ERR_CANCELED") {
          console.error("Failed to update preview plan:", error);
          setPreviewPlan(null);
        }
      }
    },
    []
  );

  const clearPreviewPlan = useCallback(() => {
    // Cancel any pending request when clearing
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setPreviewPlan(null);
  }, []);

  // Cleanup on unmount
  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  const value = useMemo(
    () => ({
      showCreateForm,
      setShowCreateForm,
      disableEdit,
      diagramContainerRef,
      previewPlan,
      goalSteps,
      toggleGoalStep,
      updatePreviewPlan,
      clearPreviewPlan,
      setGoalSteps,
    }),
    [
      showCreateForm,
      disableEdit,
      previewPlan,
      goalSteps,
      toggleGoalStep,
      updatePreviewPlan,
      clearPreviewPlan,
      setGoalSteps,
    ]
  );

  return <UIContext.Provider value={value}>{children}</UIContext.Provider>;
};

export const useUI = () => {
  const context = useContext(UIContext);
  if (context === undefined) {
    throw new Error("useUI must be used within a UIProvider");
  }
  return context;
};
