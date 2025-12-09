import React, { createContext, useCallback, useContext, useState } from "react";
import StepEditor from "../components/organisms/StepEditor";
import { Step } from "../api";

type DiagramRef = React.RefObject<HTMLDivElement | null> | undefined;

type EditorOptions = {
  step: Step | null;
  diagramContainerRef?: DiagramRef;
  onUpdate?: (step: Step) => void;
  onClose?: () => void;
};

type EditorState = EditorOptions & { open: boolean };

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
  });

  const closeEditor = useCallback(() => {
    setState((prev) => {
      prev?.onClose?.();
      return prev ? { ...prev, open: false } : prev;
    });
  }, []);

  const openEditor = useCallback((options: EditorOptions) => {
    setState({
      ...options,
      open: true,
    });
  }, []);

  const handleUpdate = useCallback(
    (updated: Step) => {
      state?.onUpdate?.(updated);
    },
    [state]
  );

  return (
    <StepEditorContext.Provider
      value={{
        openEditor,
        closeEditor,
        isOpen: state.open,
        activeStep: state.step,
      }}
    >
      {children}
      {state.open && (
        <StepEditor
          step={state.step}
          onClose={closeEditor}
          onUpdate={handleUpdate}
          diagramContainerRef={state.diagramContainerRef}
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
