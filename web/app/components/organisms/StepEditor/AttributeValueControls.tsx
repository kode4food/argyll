import React from "react";
import { SCRIPT_LANGUAGE_ARGYLL } from "@/app/api";
import DurationInput from "@/app/components/molecules/DurationInput";
import MetricSelectorInput from "@/app/components/molecules/MetricSelectorInput";
import PresetSelectInput from "@/app/components/molecules/PresetSelectInput";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import { useT } from "@/app/i18n";
import { IconAttributeFeed, IconAttributeMeta } from "@/utils/iconRegistry";
import formStyles from "./StepEditorForm.module.css";
import type { Attribute } from "./stepEditorUtils";
import { getMatchScriptPlaceholderKey } from "./stepEditorConstants";
import { useAttributeValueControls } from "./useAttributeValueControls";

interface AttributeValueControlsProps {
  attr: Attribute;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
}

const AttributeValueControls: React.FC<AttributeValueControlsProps> = ({
  attr,
  updateAttribute,
}) => {
  const t = useT();
  const { feedOptions, metaKeyOptions } = useAttributeValueControls();

  return (
    <>
      {attr.role === "meta" && (
        <PresetSelectInput
          ariaLabel={t("stepEditor.metaKeySelect")}
          className={formStyles.argValueInput}
          Icon={IconAttributeMeta}
          onChange={(v) => updateAttribute(attr.id, "metaKey", v)}
          presets={metaKeyOptions}
          placeholder={t("stepEditor.metaKeyPlaceholder")}
          value={attr.metaKey || ""}
        />
      )}
      {attr.role === "feed" && (
        <PresetSelectInput
          ariaLabel={t("stepEditor.feedSelect")}
          className={formStyles.argValueInput}
          Icon={IconAttributeFeed}
          onChange={(v) => updateAttribute(attr.id, "feedId", v)}
          presets={feedOptions}
          placeholder={t("stepEditor.feedIdPlaceholder")}
          emptyLabel={t("stepEditor.feedNoneRegistered")}
          value={attr.feedId || ""}
        />
      )}
      {attr.role === "metric" && (
        <MetricSelectorInput
          className={formStyles.argValueInput}
          language={attr.metricLanguage}
          script={attr.metricScript}
          onLanguageChange={(language) =>
            updateAttribute(attr.id, "metricLanguage", language)
          }
          onScriptChange={(script) =>
            updateAttribute(attr.id, "metricScript", script)
          }
        />
      )}
      {(attr.role === "required" || attr.role === "optional") && (
        <ScriptLanguageInlineInput
          ariaLabel={t("stepEditor.matchLanguageLabel")}
          className={formStyles.argValueInput}
          language={attr.matchLanguage || SCRIPT_LANGUAGE_ARGYLL}
          onLanguageChange={(language) =>
            updateAttribute(attr.id, "matchLanguage", language)
          }
          onScriptChange={(script) =>
            updateAttribute(attr.id, "matchScript", script)
          }
          placeholder={t(getMatchScriptPlaceholderKey(attr.matchLanguage))}
          script={attr.matchScript || ""}
          title={t("stepEditor.attributeMatchTitle")}
        />
      )}
      {(attr.role === "optional" || attr.role === "const") && (
        <input
          type="text"
          value={attr.defaultValue || ""}
          onChange={(e) =>
            updateAttribute(attr.id, "defaultValue", e.target.value)
          }
          placeholder={t("stepEditor.attributeDefaultPlaceholder")}
          className={`${formStyles.argInput} ${formStyles.argValueInput}`}
          title={t("stepEditor.attributeDefaultTitle")}
        />
      )}
      {attr.role === "optional" && (
        <DurationInput
          value={attr.deadline || 0}
          onChange={(ms) =>
            updateAttribute(attr.id, "deadline", ms || undefined)
          }
          className={`${formStyles.argInput} ${formStyles.argValueInput}`}
        />
      )}
    </>
  );
};

export default AttributeValueControls;
