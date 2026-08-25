import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { api, ExecutionPlan } from "../api";
import { useSpaces } from "../store/flowStore";

interface UIContextType {
  diagramContainerRef: React.RefObject<HTMLDivElement | null>;
  panelRef: React.RefObject<HTMLDivElement | null>;
  focusedPreviewAttribute: string | null;
  setFocusedPreviewAttribute: (attribute: string | null) => void;
  previewPlan: ExecutionPlan | null;
  setPreviewPlan: (plan: ExecutionPlan | null) => void;
  goalSteps: string[];
  toggleGoalStep: (stepId: string) => void;
  setGoalSteps: (stepIds: string[]) => void;
  spaceId: string | null;
  setSpaceId: (spaceId: string | null) => void;
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
  const [previewPlan, setPreviewPlanState] = useState<ExecutionPlan | null>(
    null
  );
  const [focusedPreviewAttribute, setFocusedPreviewAttributeState] = useState<
    string | null
  >(null);
  const [goalSteps, setGoalStepsState] = useState<string[]>([]);
  const [spaceId, setSpaceIdState] = useState<string | null>(null);
  const spaces = useSpaces();
  const diagramContainerRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  // Read by updatePreviewPlan so its identity stays stable across space changes
  const spaceIdRef = useRef<string | null>(null);

  const setPreviewPlan = useCallback((plan: ExecutionPlan | null) => {
    setPreviewPlanState(plan);
  }, []);

  const setFocusedPreviewAttribute = useCallback((attribute: string | null) => {
    setFocusedPreviewAttributeState(attribute);
  }, []);

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
        setPreviewPlanState(null);
        setFocusedPreviewAttributeState(null);
        return;
      }

      // Create new abort controller for this request
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      try {
        const plan = await api.getExecutionPlan({
          goalSteps,
          initialState,
          spaceId: spaceIdRef.current ?? undefined,
          signal: abortController.signal,
        });

        // Only update state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setPreviewPlanState(plan);
        }
      } catch (error: any) {
        // Ignore abort errors
        if (error?.name !== "AbortError" && error?.code !== "ERR_CANCELED") {
          console.error("Failed to update preview plan:", error);
          setPreviewPlanState(null);
          setFocusedPreviewAttributeState(null);
        }
      }
    },
    []
  );

  // A scope change invalidates any preview built against the previous scope
  const setSpaceId = useCallback((nextSpaceId: string | null) => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    spaceIdRef.current = nextSpaceId;
    setSpaceIdState(nextSpaceId);
    setGoalStepsState([]);
    setPreviewPlanState(null);
    setFocusedPreviewAttributeState(null);
  }, []);

  // A Space can be deleted while it is selected, leaving nothing to scope to
  useEffect(() => {
    if (spaceId && !spaces.some((space) => space.id === spaceId)) {
      setSpaceId(null);
    }
  }, [spaces, spaceId, setSpaceId]);

  const clearPreviewPlan = useCallback(() => {
    // Cancel any pending request when clearing
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setPreviewPlanState(null);
    setFocusedPreviewAttributeState(null);
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
      diagramContainerRef,
      panelRef,
      focusedPreviewAttribute,
      setFocusedPreviewAttribute,
      previewPlan,
      setPreviewPlan,
      goalSteps,
      toggleGoalStep,
      updatePreviewPlan,
      clearPreviewPlan,
      setGoalSteps,
      spaceId,
      setSpaceId,
    }),
    [
      focusedPreviewAttribute,
      setFocusedPreviewAttribute,
      previewPlan,
      setPreviewPlan,
      goalSteps,
      toggleGoalStep,
      updatePreviewPlan,
      clearPreviewPlan,
      setGoalSteps,
      spaceId,
      setSpaceId,
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
