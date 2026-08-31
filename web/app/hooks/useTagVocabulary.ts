import { useMemo } from "react";
import { useSteps } from "@/app/store/flowStore";

export const useTagVocabulary = () => {
  const steps = useSteps();

  return useMemo(() => {
    const res = new Set<string>();
    steps.forEach((step) => step.tags?.forEach((tag) => res.add(tag)));
    return [...res].sort((a, b) => a.localeCompare(b));
  }, [steps]);
};
