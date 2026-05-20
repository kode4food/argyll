import React from "react";
import { SCRIPT_LANGUAGE_LUA, StepType } from "@/app/api";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import { useT } from "@/app/i18n";
import { IconMapping } from "@/utils/iconRegistry";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { Attribute } from "./stepEditorUtils";
import { getMappingScriptPlaceholderKey } from "./stepEditorConstants";
import InlineSelectDropdown, {
  InlineSelectOption,
} from "./InlineSelectDropdown";

interface AttributeMappingPanelProps {
  attr: Attribute;
  stepType: StepType;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
  usedInputMappings: Map<string, string>;
  usedOutputMappings: Map<string, string>;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
}

const AttributeMappingPanel: React.FC<AttributeMappingPanelProps> = ({
  attr,
  stepType,
  flowInputOptions,
  flowOutputOptions,
  usedInputMappings,
  usedOutputMappings,
  updateAttribute,
}) => {
  const t = useT();
  const mappingNameHint = attr.name?.trim() || attr.id;
  const filteredFlowOutputList = flowOutputOptions.filter(
    (option) => option !== mappingNameHint
  );
  const filteredFlowInputList = flowInputOptions.filter(
    (option) => option.name !== mappingNameHint
  );

  return (
    <div className={formStyles.attrMappingPanel}>
      <span className={formStyles.mappingIndicator} aria-hidden>
        <IconMapping className={styles.iconSm} />
      </span>
      {stepType === "flow" ? (
        (() => {
          const isOutput = attr.attrType === "output";
          const flowOptions: InlineSelectOption[] = isOutput
            ? [
                { value: "", label: mappingNameHint },
                ...filteredFlowOutputList.map((opt) => ({
                  value: opt,
                  label: opt,
                  disabled:
                    usedOutputMappings.has(opt) &&
                    usedOutputMappings.get(opt) !== attr.id,
                })),
              ]
            : [
                { value: "", label: mappingNameHint },
                ...filteredFlowInputList.map((opt) => ({
                  value: opt.name,
                  label: opt.name,
                  highlight: opt.required,
                  disabled:
                    usedInputMappings.has(opt.name) &&
                    usedInputMappings.get(opt.name) !== attr.id,
                })),
              ];
          return (
            <InlineSelectDropdown
              value={attr.mappingName || ""}
              options={flowOptions}
              onChange={(v) => updateAttribute(attr.id, "mappingName", v)}
              className={formStyles.mappingInlineInput}
              disabled={
                isOutput
                  ? flowOutputOptions.length === 0
                  : flowInputOptions.length === 0
              }
            />
          );
        })()
      ) : (
        <input
          type="text"
          value={attr.mappingName || ""}
          onChange={(e) =>
            updateAttribute(attr.id, "mappingName", e.target.value)
          }
          placeholder={mappingNameHint}
          className={`${formStyles.formControl} ${formStyles.mappingInlineInput}`}
        />
      )}
      <ScriptLanguageInlineInput
        ariaLabel={t("stepEditor.mappingLanguageLabel")}
        className={formStyles.mappingScriptInlineInput}
        language={attr.mappingLanguage || SCRIPT_LANGUAGE_LUA}
        onLanguageChange={(language) =>
          updateAttribute(attr.id, "mappingLanguage", language)
        }
        onScriptChange={(script) =>
          updateAttribute(attr.id, "mappingScript", script)
        }
        placeholder={t(getMappingScriptPlaceholderKey(attr.mappingLanguage))}
        script={attr.mappingScript || ""}
      />
    </div>
  );
};

export default AttributeMappingPanel;
