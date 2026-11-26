import React, {
  createContext,
  useCallback,
  useContext,
  useState,
  useMemo,
} from "react";
import StepEditor from "@/app/components/organisms/StepEditor";
import { Step } from "@/app/api";

type DiagramRef = React.RefObject<HTMLDivElement | null> | undefined;

type EditorOptions = {
  step: Step | null;
  diagramContainerRef?: DiagramRef;
  onUpdate?: (step: Step) => void;
  onClose?: () => void;
  draft?: Step;
};

type EditorState = EditorOptions & { open: boolean; serial: number };

type StepEditorContextValue = {
  openEditor: (options: EditorOptions) => void;
  closeEditor: () => void;
  isOpen: boolean;
  activeStep: Step | null;
};

const StepEditorContext = createContext<StepEditorContextValue | null>(null);

export const StepEditorProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [state, setState] = useState<EditorState>({
    step: null,
    open: false,
    serial: 0,
  });

  const closeEditor = useCallback(() => {
    setState((prev) => {
      if (!prev || !prev.open) return prev;
      prev.onClose?.();
      return { ...prev, open: false };
    });
  }, []);

  const openEditor = useCallback((options: EditorOptions) => {
    setState((prev) => ({
      ...options,
      open: true,
      serial: prev.serial + 1,
    }));
  }, []);

  const handleUpdate = useCallback(
    (updated: Step) => {
      state?.onUpdate?.(updated);
    },
    [state]
  );

  const contextValue = useMemo(
    () => ({
      openEditor,
      closeEditor,
      isOpen: state.open,
      activeStep: state.step,
    }),
    [openEditor, closeEditor, state.open, state.step]
  );

  return (
    <StepEditorContext.Provider value={contextValue}>
      {children}
      {state.open && (
        <StepEditor
          key={state.serial}
          step={state.step}
          onClose={closeEditor}
          onUpdate={handleUpdate}
          diagramContainerRef={state.diagramContainerRef}
          draft={state.draft}
        />
      )}
    </StepEditorContext.Provider>
  );
};

export const useStepEditorContext = (): StepEditorContextValue => {
  const ctx = useContext(StepEditorContext);
  if (!ctx) {
    throw new Error(
      "useStepEditorContext must be used within a StepEditorProvider"
    );
  }
  return ctx;
};
