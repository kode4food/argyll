import React from "react";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import PresetSelectInput from "@/app/components/molecules/PresetSelectInput";
import { useT } from "@/app/i18n";
import { IconAttributeMetric } from "@/utils/iconRegistry";
import {
  MetricSelectorInputOptions,
  useMetricSelectorInput,
} from "./useMetricSelectorInput";

export interface MetricSelectorInputProps extends MetricSelectorInputOptions {
  className?: string;
}

const MetricSelectorInput: React.FC<MetricSelectorInputProps> = ({
  className,
  ...options
}) => {
  const t = useT();
  const { isCustom, language, script, presets, handlePresetChange } =
    useMetricSelectorInput(options);

  const renderCustom = (picker: React.ReactNode) => (
    <ScriptLanguageInlineInput
      ariaLabel={t("stepEditor.metricLanguageLabel")}
      className={className}
      language={language}
      leading={picker}
      onLanguageChange={options.onLanguageChange}
      onScriptChange={options.onScriptChange}
      placeholder={t("stepEditor.metricSelectorPlaceholder")}
      script={script}
      title={t("stepEditor.metricSelectorTitle")}
    />
  );

  return (
    <PresetSelectInput
      ariaLabel={t("stepEditor.metricPresetSelect")}
      className={className}
      Icon={IconAttributeMetric}
      onChange={handlePresetChange}
      presets={presets}
      placeholder={t("stepEditor.metricPresetPlaceholder")}
      value={script}
      customLabel={t("stepEditor.metricSelector.custom")}
      renderCustom={isCustom ? renderCustom : undefined}
    />
  );
};

export default MetricSelectorInput;
