import React from "react";
import {
  AttributeType,
  InputCollect,
  SCRIPT_LANGUAGE_JPATH,
  StepType,
} from "@/app/api";
import DurationInput from "@/app/components/molecules/DurationInput";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import { useT } from "@/app/i18n";
import {
  IconArrayMultiple,
  IconArraySingle,
  IconExpandDown,
  IconExpandUp,
  IconRemove,
} from "@/utils/iconRegistry";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { Attribute, getAttributeIconProps } from "./stepEditorUtils";
import {
  ATTRIBUTE_TYPES,
  getMatchScriptPlaceholderKey,
  INPUT_COLLECT_TYPES,
} from "./stepEditorConstants";
import AttributeMappingPanel from "./AttributeMappingPanel";

interface AttributeRowProps {
  attr: Attribute;
  stepType: StepType;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
  usedInputMappings: Map<string, string>;
  usedOutputMappings: Map<string, string>;
  cycleAttributeType: (
    id: string,
    currentType: "input" | "optional" | "const" | "output"
  ) => void;
  cycleInputCollect: (id: string, currentCollect?: InputCollect) => void;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
  removeAttribute: (id: string) => void;
}

const AttributeRow: React.FC<AttributeRowProps> = ({
  attr,
  stepType,
  flowInputOptions,
  flowOutputOptions,
  usedInputMappings,
  usedOutputMappings,
  cycleAttributeType,
  cycleInputCollect,
  updateAttribute,
  removeAttribute,
}) => {
  const t = useT();
  const [isMappingExpanded, setIsMappingExpanded] = React.useState(false);

  const collect =
    attr.collect && INPUT_COLLECT_TYPES.includes(attr.collect)
      ? attr.collect
      : "first";
  const canCollect = attr.attrType === "input" || attr.attrType === "optional";
  const hasMappingConfigured = Boolean(
    attr.mappingName?.trim() || attr.mappingScript?.trim()
  );

  return (
    <div className={formStyles.attrRow}>
      <div className={formStyles.attrRowInputs}>
        <button
          type="button"
          onClick={() => cycleAttributeType(attr.id, attr.attrType)}
          className={`${formStyles.iconButton} ${formStyles.attrIconButtonStyle}`}
          title={t("stepEditor.cycleAttributeType", { type: attr.attrType })}
        >
          {(() => {
            const { Icon, className } = getAttributeIconProps(attr.attrType);
            return <Icon className={`${styles.iconMd} ${className}`} />;
          })()}
        </button>
        <select
          value={attr.dataType}
          onChange={(e) => updateAttribute(attr.id, "dataType", e.target.value)}
          className={formStyles.argType}
        >
          {ATTRIBUTE_TYPES.map((type) => (
            <option key={type} value={type}>
              {type}
            </option>
          ))}
        </select>
        <input
          type="text"
          value={attr.name}
          onChange={(e) => updateAttribute(attr.id, "name", e.target.value)}
          placeholder={t("stepEditor.attributeNamePlaceholder")}
          className={`${formStyles.argInput} ${formStyles.argNameInput}`}
        />
        {attr.attrType === "input" && (
          <ScriptLanguageInlineInput
            ariaLabel={t("stepEditor.matchLanguageLabel")}
            className={formStyles.argValueInput}
            language={attr.matchLanguage || SCRIPT_LANGUAGE_JPATH}
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
        {(attr.attrType === "optional" || attr.attrType === "const") && (
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
        {attr.attrType === "optional" && (
          <DurationInput
            value={attr.deadline || 0}
            onChange={(ms) =>
              updateAttribute(attr.id, "deadline", ms || undefined)
            }
            className={formStyles.argInput}
          />
        )}
        {attr.attrType !== "output" &&
          attr.dataType === AttributeType.Array && (
            <div className={formStyles.forEachToggleGroup}>
              <button
                type="button"
                onClick={(e) => {
                  updateAttribute(attr.id, "forEach", false);
                  e.currentTarget.blur();
                }}
                className={`${formStyles.forEachToggle} ${
                  !attr.forEach ? formStyles.forEachToggleActive : ""
                }`}
                title={t("stepEditor.arraySingleTitle")}
              >
                <IconArraySingle className={styles.iconSm} />
                <span>{t("stepEditor.arraySingleLabel")}</span>
              </button>
              <button
                type="button"
                onClick={(e) => {
                  updateAttribute(attr.id, "forEach", true);
                  e.currentTarget.blur();
                }}
                className={`${formStyles.forEachToggle} ${
                  attr.forEach ? formStyles.forEachToggleActive : ""
                }`}
                title={t("stepEditor.arrayMultiTitle")}
              >
                <IconArrayMultiple className={styles.iconSm} />
                <span>{t("stepEditor.arrayMultiLabel")}</span>
              </button>
            </div>
          )}
        {canCollect && (
          <button
            type="button"
            onClick={(e) => {
              cycleInputCollect(attr.id, collect);
              e.currentTarget.blur();
            }}
            className={`${formStyles.iconButton} ${formStyles.collectButtonStyle}`}
            title={t("stepEditor.cycleInputCollect", { collect })}
            aria-label={t("stepEditor.cycleInputCollect", { collect })}
          >
            <span
              className={formStyles.collectIcon}
              style={{
                maskImage: `url(/icons/collect-${collect}.svg)`,
                WebkitMaskImage: `url(/icons/collect-${collect}.svg)`,
              }}
            />
          </button>
        )}
        {attr.attrType !== "const" && (
          <button
            type="button"
            onClick={() => setIsMappingExpanded((current) => !current)}
            className={`${formStyles.iconButton} ${formStyles.mappingExpandButton} ${
              hasMappingConfigured ? formStyles.mappingExpandButtonActive : ""
            }`}
            title={t("stepEditor.mappingLabel")}
            aria-label={`${t("stepEditor.mappingLabel")} ${
              attr.name || attr.id
            }`}
          >
            {isMappingExpanded ? (
              <IconExpandUp className={styles.iconSm} />
            ) : (
              <IconExpandDown className={styles.iconSm} />
            )}
          </button>
        )}
        <button
          onClick={() => removeAttribute(attr.id)}
          className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
          title={t("stepEditor.removeAttribute")}
        >
          <IconRemove className={styles.iconSm} />
        </button>
      </div>
      {isMappingExpanded && attr.attrType !== "const" && (
        <AttributeMappingPanel
          attr={attr}
          stepType={stepType}
          flowInputOptions={flowInputOptions}
          flowOutputOptions={flowOutputOptions}
          usedInputMappings={usedInputMappings}
          usedOutputMappings={usedOutputMappings}
          updateAttribute={updateAttribute}
        />
      )}
      {attr.validationError && (
        <div className={formStyles.attrValidationError}>
          {attr.validationError}
        </div>
      )}
    </div>
  );
};

export default AttributeRow;
