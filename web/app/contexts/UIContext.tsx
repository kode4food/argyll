"use client";

import React, {
  createContext,
  useContext,
  useState,
  useRef,
  useCallback,
  useMemo,
} from "react";
import { ExecutionPlan, api } from "../api";

interface UIContextType {
  showCreateForm: boolean;
  setShowCreateForm: (show: boolean) => void;
  disableEdit: boolean;
  diagramContainerRef: React.RefObject<HTMLDivElement | null>;
  previewPlan: ExecutionPlan | null;
  selectedStep: string | null;
  goalStepIds: string[];
  updatePreviewPlan: (
    goalStepIds: string[],
    initialState: Record<string, any>
  ) => Promise<void>;
  clearPreviewPlan: () => void;
  setSelectedStep: (stepId: string | null) => void;
  setGoalStepIds: (stepIds: string[]) => void;
}

const UIContext = createContext<UIContextType | undefined>(undefined);

export const UIProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [previewPlan, setPreviewPlan] = useState<ExecutionPlan | null>(null);
  const [selectedStep, setSelectedStep] = useState<string | null>(null);
  const [goalStepIds, setGoalStepIds] = useState<string[]>([]);
  const diagramContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const disableEdit = showCreateForm;

  const updatePreviewPlan = useCallback(
    async (goalStepIds: string[], initialState: Record<string, any>) => {
      // Cancel any pending request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      if (goalStepIds.length === 0) {
        setPreviewPlan(null);
        return;
      }

      // Create new abort controller for this request
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      try {
        const plan = await api.getExecutionPlan(
          goalStepIds,
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
      selectedStep,
      goalStepIds,
      updatePreviewPlan,
      clearPreviewPlan,
      setSelectedStep,
      setGoalStepIds,
    }),
    [
      showCreateForm,
      disableEdit,
      previewPlan,
      selectedStep,
      goalStepIds,
      updatePreviewPlan,
      clearPreviewPlan,
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
