import React from "react";
import { METRIC_SELECTORS, SCRIPT_LANGUAGE_ARGYLL } from "@/app/api";
import { useT } from "@/app/i18n";
import { PRESET_CUSTOM } from "@/app/components/molecules/PresetSelectInput";

export interface MetricSelectorInputOptions {
  language?: string;
  script?: string;
  onLanguageChange: (language: string) => void;
  onScriptChange: (script: string) => void;
}

// A blank selector starts on the presets, so a language only appears once Custom is chosen
const isPresetSelector = (language: string, script: string): boolean => {
  const isPresetScript = METRIC_SELECTORS.some((sel) => sel === script);
  return language === SCRIPT_LANGUAGE_ARGYLL && (!script || isPresetScript);
};

export const useMetricSelectorInput = ({
  language: rawLanguage,
  script: rawScript,
  onLanguageChange,
  onScriptChange,
}: MetricSelectorInputOptions) => {
  const t = useT();
  const script = rawScript || "";
  const language = rawLanguage || SCRIPT_LANGUAGE_ARGYLL;
  const [isCustom, setIsCustom] = React.useState(
    () => !isPresetSelector(language, script)
  );

  const presets = METRIC_SELECTORS.map((sel) => ({
    value: sel,
    label: t(`stepEditor.metricSelector.${sel}`),
  }));

  const handlePresetChange = (value: string) => {
    setIsCustom(value === PRESET_CUSTOM);
    if (value === PRESET_CUSTOM) return;
    onLanguageChange(SCRIPT_LANGUAGE_ARGYLL);
    onScriptChange(value);
  };

  return { isCustom, language, script, presets, handlePresetChange };
};
