import React from "react";
import { METRIC_SELECTS, SCRIPT_LANGUAGE_ARGYLL } from "@/app/api";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import { useT } from "@/app/i18n";
import { IconAttributeMetric } from "@/utils/iconRegistry";
import formStyles from "./StepEditorForm.module.css";
import PresetSelectInput, { PRESET_CUSTOM } from "./PresetSelectInput";
import type { Attribute } from "./stepEditorUtils";

export interface MetricSelectInputProps {
  attr: Attribute;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
}

// A blank select starts on the presets, so a language only appears once Custom is chosen
const isPresetSelect = (attr: Attribute): boolean => {
  const script = attr.metricScript || "";
  const language = attr.metricLanguage || SCRIPT_LANGUAGE_ARGYLL;
  const isPresetScript = METRIC_SELECTS.some((sel) => sel === script);
  return language === SCRIPT_LANGUAGE_ARGYLL && (!script || isPresetScript);
};

const MetricSelectInput: React.FC<MetricSelectInputProps> = ({
  attr,
  updateAttribute,
}) => {
  const t = useT();
  const [isCustom, setIsCustom] = React.useState(() => !isPresetSelect(attr));
  const script = attr.metricScript || "";

  const presets = METRIC_SELECTS.map((sel) => ({
    value: sel,
    label: t(`stepEditor.metricSelect.${sel}`),
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
      placeholder={t("stepEditor.metricSelectPlaceholder")}
      script={script}
      title={t("stepEditor.metricSelectTitle")}
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
      customLabel={t("stepEditor.metricSelect.custom")}
      renderCustom={isCustom ? renderCustom : undefined}
    />
  );
};

export default MetricSelectInput;
