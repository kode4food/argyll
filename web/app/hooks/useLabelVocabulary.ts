import { useCallback, useMemo } from "react";
import { useSteps } from "@/app/store/flowStore";

export const useLabelVocabulary = () => {
  const steps = useSteps();

  const byKey = useMemo(() => {
    const res = new Map<string, Set<string>>();
    steps.forEach((step) =>
      Object.entries(step.labels ?? {}).forEach(([key, value]) => {
        const values = res.get(key) ?? new Set<string>();
        values.add(value);
        res.set(key, values);
      })
    );
    return res;
  }, [steps]);

  const labelKeys = useMemo(
    () => [...byKey.keys()].sort((a, b) => a.localeCompare(b)),
    [byKey]
  );

  const valuesForKey = useCallback(
    (key: string) =>
      [...(byKey.get(key) ?? [])].sort((a, b) => a.localeCompare(b)),
    [byKey]
  );

  return { labelKeys, valuesForKey };
};
