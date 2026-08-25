import { useCallback, useRef, useState } from "react";
import { Step } from "@/app/api";
import { Label } from "./stepEditorTypes";
import { buildLabelRows, buildLabelsFromStep } from "./stepLabelUtils";

export function useLabelList(
  step: Step | null,
  defaults?: Record<string, string>
) {
  const [labels, setLabels] = useState<Label[]>(() =>
    step ? buildLabelsFromStep(step) : buildLabelRows(defaults)
  );
  const labelCounterRef = useRef(0);

  const addLabel = useCallback(() => {
    setLabels((current) => [
      ...current,
      { id: `label-${++labelCounterRef.current}`, key: "", value: "" },
    ]);
  }, []);

  const updateLabel = useCallback(
    (id: string, field: "key" | "value", value: string) => {
      setLabels((current) =>
        current.map((label) =>
          label.id === id ? { ...label, [field]: value } : label
        )
      );
    },
    []
  );

  const removeLabel = useCallback((id: string) => {
    setLabels((current) => current.filter((label) => label.id !== id));
  }, []);

  const resetLabels = useCallback((nextStep: Step | null) => {
    setLabels(buildLabelsFromStep(nextStep));
  }, []);

  return { labels, addLabel, updateLabel, removeLabel, resetLabels };
}
