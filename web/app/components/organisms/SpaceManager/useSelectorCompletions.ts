import React from "react";
import { SCRIPT_LANGUAGE_ARGYLL, ScriptConfig } from "@/app/api";
import { useSteps } from "@/app/store/flowStore";
import { spaceSelectorCompletions } from "@/utils/scriptCompletions";

export const useSelectorCompletions = (selector?: ScriptConfig) => {
  const steps = useSteps();
  const language = selector?.language ?? SCRIPT_LANGUAGE_ARGYLL;
  const completions = React.useMemo(
    () => spaceSelectorCompletions(steps, language),
    [language, steps]
  );
  return { language, completions };
};
