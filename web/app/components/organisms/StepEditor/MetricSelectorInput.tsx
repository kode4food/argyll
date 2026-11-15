import React from "react";
import { METRIC_SELECTORS, SCRIPT_LANGUAGE_ARGYLL } from "@/app/api";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import { useT } from "@/app/i18n";
import { IconAttributeMetric } from "@/utils/iconRegistry";
import formStyles from "./StepEditorForm.module.css";
import PresetSelectInput, { PRESET_CUSTOM } from "./PresetSelectInput";
import type { Attribute } from "./stepEditorUtils";

export interface MetricSelectorInputProps {
  attr: Attribute;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
}

// A blank selector starts on the presets, so a language only appears once Custom is chosen
const isPresetSelector = (attr: Attribute): boolean => {
  const script = attr.metricScript || "";
  const language = attr.metricLanguage || SCRIPT_LANGUAGE_ARGYLL;
  const isPresetScript = METRIC_SELECTORS.some((sel) => sel === script);
  return language === SCRIPT_LANGUAGE_ARGYLL && (!script || isPresetScript);
};

const MetricSelectorInput: React.FC<MetricSelectorInputProps> = ({
  attr,
  updateAttribute,
}) => {
  const t = useT();
  const [isCustom, setIsCustom] = React.useState(() => !isPresetSelector(attr));
  const script = attr.metricScript || "";

  const presets = METRIC_SELECTORS.map((sel) => ({
    value: sel,
    label: t(`stepEditor.metricSelector.${sel}`),
  }));

  const handlePresetChange = (value: string) => {
    setIsCustom(value === PRESET_CUSTOM);
    if (value === PRESET_CUSTOM) return;
    updateAttribute(attr.id, "metricLanguage", SCRIPT_LANGUAGE_ARGYLL);
    updateAttribute(attr.id, "metricScript", value);
  };

  const renderCustom = (picker: React.ReactNode) => (
    <ScriptLanguageInlineInput
      ariaLabel={t("stepEditor.metricLanguageLabel")}
      className={formStyles.argValueInput}
      language={attr.metricLanguage || SCRIPT_LANGUAGE_ARGYLL}
      leading={picker}
      onLanguageChange={(language) =>
        updateAttribute(attr.id, "metricLanguage", language)
      }
      onScriptChange={(value) =>
        updateAttribute(attr.id, "metricScript", value)
      }
      placeholder={t("stepEditor.metricSelectorPlaceholder")}
      script={script}
      title={t("stepEditor.metricSelectorTitle")}
    />
  );

  return (
    <PresetSelectInput
      ariaLabel={t("stepEditor.metricPresetSelect")}
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
