import React from "react";
import {
  AttributeType,
  InputCollect,
  SCRIPT_LANGUAGE_LUA,
  StepType,
} from "@/app/api";
import DurationInput from "@/app/components/molecules/DurationInput";
import { useT } from "@/app/i18n";
import {
  IconAdd,
  IconArrayMultiple,
  IconArraySingle,
  IconExpandDown,
  IconExpandUp,
  IconMapping,
  IconRemove,
} from "@/utils/iconRegistry";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { Attribute, getAttributeIconProps } from "./stepEditorUtils";
import {
  ATTRIBUTE_TYPES,
  getMappingScriptPlaceholderKey,
  INPUT_COLLECT_TYPES,
  MAPPING_LANGUAGE_OPTIONS,
} from "./stepEditorConstants";

interface StepEditorAttributesSectionProps {
  addAttribute: () => void;
  attributes: Attribute[];
  cycleAttributeType: (
    id: string,
    currentType: "input" | "optional" | "const" | "output"
  ) => void;
  cycleInputCollect: (id: string, currentCollect?: InputCollect) => void;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
  removeAttribute: (id: string) => void;
  stepType: StepType;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
}

const StepEditorAttributesSection: React.FC<
  StepEditorAttributesSectionProps
> = ({
  addAttribute,
  attributes,
  cycleAttributeType,
  cycleInputCollect,
  flowInputOptions,
  flowOutputOptions,
  removeAttribute,
  stepType,
  updateAttribute,
}) => {
  const t = useT();
  const [expandedMappingAttributeID, setExpandedMappingAttributeID] =
    React.useState<string | null>(null);

  const usedInputMappings = new Map<string, string>();
  const usedOutputMappings = new Map<string, string>();

  attributes.forEach((attr) => {
    const mappingName = attr.mappingName?.trim();
    if (!mappingName) {
      return;
    }
    if (attr.attrType === "output") {
      usedOutputMappings.set(mappingName, attr.id);
      return;
    }
    usedInputMappings.set(mappingName, attr.id);
  });

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.attributesLabel")}
        </label>
        <button
          onClick={addAttribute}
          className={`${formStyles.iconButton} ${formStyles.addButtonStyle}`}
          title={t("stepEditor.addAttribute")}
        >
          <IconAdd className={styles.iconMd} />
        </button>
      </div>
      <div className={formStyles.argList}>
        {attributes.length === 0 && (
          <div
            className={`${formStyles.attrRow} ${formStyles.attrPlaceholder}`}
            aria-hidden
          >
            <div className={formStyles.attrRowInputs}>
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderIcon}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSelect}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderInput}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSmall}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderButton}`}
              />
            </div>
            <div className={formStyles.placeholderHint}>
              {t("stepEditor.attributesHint")}
            </div>
          </div>
        )}
        {attributes.map((attr) => {
          const collect =
            attr.collect && INPUT_COLLECT_TYPES.includes(attr.collect)
              ? attr.collect
              : "first";
          const canCollect =
            attr.attrType === "input" || attr.attrType === "optional";
          const isMappingExpanded =
            expandedMappingAttributeID === attr.id && attr.attrType !== "const";
          const hasMappingConfigured = Boolean(
            attr.mappingName?.trim() || attr.mappingScript?.trim()
          );
          const mappingNameHint = attr.name?.trim() || attr.id;
          const filteredFlowOutputList = flowOutputOptions.filter(
            (option) => option !== mappingNameHint
          );
          const filteredFlowInputList = flowInputOptions.filter(
            (option) => option.name !== mappingNameHint
          );

          return (
            <div key={attr.id} className={formStyles.attrRow}>
              <div className={formStyles.attrRowInputs}>
                <button
                  type="button"
                  onClick={() => cycleAttributeType(attr.id, attr.attrType)}
                  className={`${formStyles.iconButton} ${formStyles.attrIconButtonStyle}`}
                  title={t("stepEditor.cycleAttributeType", {
                    type: attr.attrType,
                  })}
                >
                  {(() => {
                    const { Icon, className } = getAttributeIconProps(
                      attr.attrType
                    );
                    return <Icon className={`${styles.iconMd} ${className}`} />;
                  })()}
                </button>
                <select
                  value={attr.dataType}
                  onChange={(e) =>
                    updateAttribute(attr.id, "dataType", e.target.value)
                  }
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
                  onChange={(e) =>
                    updateAttribute(attr.id, "name", e.target.value)
                  }
                  placeholder={t("stepEditor.attributeNamePlaceholder")}
                  className={`${formStyles.argInput} ${formStyles.argNameInput}`}
                />
                {(attr.attrType === "optional" ||
                  attr.attrType === "const") && (
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
                    value={attr.timeout || 0}
                    onChange={(ms) =>
                      updateAttribute(attr.id, "timeout", ms || undefined)
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
                        className={`${formStyles.forEachToggle} ${!attr.forEach ? formStyles.forEachToggleActive : ""}`}
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
                        className={`${formStyles.forEachToggle} ${attr.forEach ? formStyles.forEachToggleActive : ""}`}
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
                    onClick={() =>
                      setExpandedMappingAttributeID((current) =>
                        current === attr.id ? null : attr.id
                      )
                    }
                    className={`${formStyles.iconButton} ${formStyles.mappingExpandButton} ${
                      hasMappingConfigured
                        ? formStyles.mappingExpandButtonActive
                        : ""
                    }`}
                    title={t("stepEditor.mappingLabel")}
                    aria-label={`${t("stepEditor.mappingLabel")} ${attr.name || attr.id}`}
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
              {isMappingExpanded && (
                <div className={formStyles.attrMappingPanel}>
                  <span className={formStyles.mappingIndicator} aria-hidden>
                    <IconMapping className={styles.iconSm} />
                  </span>
                  {stepType === "flow" ? (
                    <select
                      value={attr.mappingName || ""}
                      onChange={(e) =>
                        updateAttribute(attr.id, "mappingName", e.target.value)
                      }
                      className={`${formStyles.flowMapSelect} ${formStyles.mappingInlineInput} ${formStyles.mappingInlineSelect}`}
                      disabled={
                        attr.attrType === "output"
                          ? flowOutputOptions.length === 0
                          : flowInputOptions.length === 0
                      }
                    >
                      <option value="">{mappingNameHint}</option>
                      {attr.attrType === "output"
                        ? filteredFlowOutputList.map((option) => (
                            <option
                              key={option}
                              value={option}
                              disabled={
                                usedOutputMappings.has(option) &&
                                usedOutputMappings.get(option) !== attr.id
                              }
                            >
                              {option}
                            </option>
                          ))
                        : filteredFlowInputList.map((option) => (
                            <option
                              key={option.name}
                              value={option.name}
                              disabled={
                                usedInputMappings.has(option.name) &&
                                usedInputMappings.get(option.name) !== attr.id
                              }
                              className={
                                option.required
                                  ? formStyles.flowMapOptionRequired
                                  : undefined
                              }
                            >
                              {option.name}
                            </option>
                          ))}
                    </select>
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
                  <div
                    className={formStyles.languageSelectorGroup}
                    aria-label={t("stepEditor.mappingLanguageLabel")}
                  >
                    {MAPPING_LANGUAGE_OPTIONS.map((option) => (
                      <button
                        key={option.value}
                        type="button"
                        onClick={(e) => {
                          updateAttribute(
                            attr.id,
                            "mappingLanguage",
                            option.value
                          );
                          e.currentTarget.blur();
                        }}
                        className={`${formStyles.languageButton} ${
                          (attr.mappingLanguage || SCRIPT_LANGUAGE_LUA) ===
                          option.value
                            ? formStyles.languageButtonActive
                            : ""
                        }`}
                        title={t(option.labelKey)}
                      >
                        {t(option.labelKey)}
                      </button>
                    ))}
                  </div>
                  <input
                    type="text"
                    value={attr.mappingScript || ""}
                    onChange={(e) =>
                      updateAttribute(attr.id, "mappingScript", e.target.value)
                    }
                    className={`${formStyles.formControl} ${formStyles.mappingScriptInlineInput}`}
                    placeholder={t(
                      getMappingScriptPlaceholderKey(attr.mappingLanguage)
                    )}
                  />
                </div>
              )}
              {attr.validationError && (
                <div className={formStyles.attrValidationError}>
                  {attr.validationError}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default StepEditorAttributesSection;
