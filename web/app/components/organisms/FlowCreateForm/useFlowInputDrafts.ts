import React from "react";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import { formatJSON } from "@/utils/jsonUtils";
import {
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  isAtDefaultValue,
} from "./flowFormUtils";

export type EditorMode = "basic" | "json";

export interface FlowInputDraftsOptions {
  flowInputOptions: FlowInputOption[];
  initialState: string;
  setInitialState: (value: string) => void;
  focusedPreviewAttribute: string | null;
  setFocusedPreviewAttribute: (name: string | null) => void;
}

export const useFlowInputDrafts = ({
  flowInputOptions,
  initialState,
  setInitialState,
  focusedPreviewAttribute,
  setFocusedPreviewAttribute,
}: FlowInputDraftsOptions) => {
  const [editorMode, setEditorMode] = React.useState<EditorMode>("basic");
  const [flowInputDraftValues, setFlowInputDraftValues] = React.useState<
    Record<string, string>
  >({});

  React.useEffect(() => {
    if (editorMode !== "basic") {
      setFocusedPreviewAttribute(null);
      setFlowInputDraftValues({});
    }
  }, [editorMode, setFocusedPreviewAttribute]);

  const flowInputNames = React.useMemo(
    () => flowInputOptions.map((option) => option.name),
    [flowInputOptions]
  );

  React.useEffect(() => {
    if (editorMode !== "basic") {
      return;
    }

    setFlowInputDraftValues((current) => {
      const nextValues: Record<string, string> = {};
      flowInputNames.forEach((name) => {
        if (current[name] !== undefined) {
          nextValues[name] = current[name];
        }
      });
      return nextValues;
    });

    if (
      focusedPreviewAttribute &&
      !flowInputNames.includes(focusedPreviewAttribute)
    ) {
      setFocusedPreviewAttribute(null);
    }
  }, [
    editorMode,
    flowInputNames,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  ]);
  const flowInputValuesRaw = React.useMemo(
    () => buildInitialStateInputValues(initialState, flowInputNames),
    [flowInputNames, initialState]
  );
  const flowInputValues = React.useMemo(() => {
    const values: Record<string, string> = {};

    flowInputOptions.forEach((option) => {
      const rawValue = flowInputValuesRaw[option.name] || "";
      const draftValue = flowInputDraftValues[option.name];
      values[option.name] =
        draftValue ?? (isAtDefaultValue(option, rawValue) ? "" : rawValue);
    });

    return values;
  }, [flowInputDraftValues, flowInputOptions, flowInputValuesRaw]);

  const commitBasicInputValues = React.useCallback(() => {
    const nextValues = {
      ...flowInputValuesRaw,
      ...flowInputDraftValues,
    };
    const nextState = buildInitialStateFromInputValues(
      nextValues,
      flowInputNames
    );

    setInitialState(formatJSON(nextState));
    setFlowInputDraftValues({});
  }, [
    flowInputDraftValues,
    flowInputNames,
    flowInputValuesRaw,
    setInitialState,
  ]);

  const handleBasicInputChange = React.useCallback(
    (name: string, value: string) => {
      setFlowInputDraftValues((current) => ({
        ...current,
        [name]: value,
      }));
    },
    []
  );

  const handleEditorModeChange = React.useCallback(
    (mode: EditorMode) => {
      if (editorMode === "basic" && mode === "json") {
        commitBasicInputValues();
      }
      setEditorMode(mode);
    },
    [commitBasicInputValues, editorMode]
  );

  return {
    editorMode,
    flowInputValues,
    flowInputValuesRaw,
    commitBasicInputValues,
    handleBasicInputChange,
    handleEditorModeChange,
  };
};
