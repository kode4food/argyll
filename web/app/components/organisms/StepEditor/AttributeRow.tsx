import React from "react";
import {
  AttributeType,
  META_KEYS,
  SCRIPT_LANGUAGE_JPATH,
  StepType,
} from "@/app/api";
import DurationInput from "@/app/components/molecules/DurationInput";
import ScriptLanguageInlineInput from "@/app/components/molecules/ScriptLanguageInlineInput";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
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
  ATTRIBUTE_ROLE_TYPES,
  ATTRIBUTE_TYPES,
  getMatchScriptPlaceholderKey,
  INPUT_COLLECT_TYPES,
} from "./stepEditorConstants";
import AttributeMappingPanel from "./AttributeMappingPanel";
import ComboInput from "./ComboInput";
import IconDropdown, { IconDropdownOption } from "./IconDropdown";
import InlineSelectDropdown, {
  InlineSelectOption,
} from "./InlineSelectDropdown";

interface AttributeRowProps {
  attr: Attribute;
  stepType: StepType;
  flowInputOptions: FlowInputOption[];
  flowOutputOptions: string[];
  usedInputMappings: Map<string, string>;
  usedOutputMappings: Map<string, string>;
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
  updateAttribute,
  removeAttribute,
}) => {
  const t = useT();
  const [isMappingExpanded, setIsMappingExpanded] = React.useState(false);

  const collect =
    attr.collect && INPUT_COLLECT_TYPES.includes(attr.collect)
      ? attr.collect
      : "first";
  const canCollect = attr.role === "required" || attr.role === "optional";
  const hasMappingConfigured = Boolean(
    attr.mappingName?.trim() || attr.mappingScript?.trim()
  );

  const roleOptions: IconDropdownOption[] = ATTRIBUTE_ROLE_TYPES.map((type) => {
    const { Icon, className } = getAttributeIconProps(type);
    return {
      value: type,
      label: t(`stepEditor.attrRole.${type}`),
      icon: <Icon className={`${styles.iconSm} ${className}`} />,
    };
  });

  const collectOptions: IconDropdownOption[] = INPUT_COLLECT_TYPES.map((c) => ({
    value: c,
    label: t(`stepEditor.collect.${c}`),
    icon: (
      <span
        className={formStyles.collectIcon}
        style={{
          maskImage: `url(/icons/collect-${c}.svg)`,
          WebkitMaskImage: `url(/icons/collect-${c}.svg)`,
        }}
      />
    ),
  }));

  const dataTypeOptions: InlineSelectOption[] = ATTRIBUTE_TYPES.map((type) => ({
    value: type,
    label: type,
  }));

  const { Icon: RoleIcon, className: roleClassName } = getAttributeIconProps(
    attr.role
  );

  const showForEach =
    attr.role !== "output" &&
    attr.role !== "meta" &&
    attr.dataType === AttributeType.Array;

  const hasValueControl = attr.role !== "output";
  const nameClass = hasValueControl
    ? formStyles.argNameInputFixed
    : formStyles.argNameInput;

  const typeControl = (
    <SegmentedGroup className={formStyles.argType}>
      <InlineSelectDropdown
        value={attr.dataType}
        options={dataTypeOptions}
        onChange={(v) => updateAttribute(attr.id, "dataType", v)}
      />
      {showForEach && (
        <IconDropdown
          ariaLabel={t("stepEditor.forEachSelect")}
          faceIcon={
            attr.forEach ? (
              <IconArrayMultiple className={styles.iconSm} />
            ) : (
              <IconArraySingle className={styles.iconSm} />
            )
          }
          onChange={(v) => updateAttribute(attr.id, "forEach", v === "multi")}
          options={[
            {
              value: "single",
              label: t("stepEditor.arraySingleLabel"),
              icon: <IconArraySingle className={styles.iconSm} />,
            },
            {
              value: "multi",
              label: t("stepEditor.arrayMultiLabel"),
              icon: <IconArrayMultiple className={styles.iconSm} />,
            },
          ]}
          value={attr.forEach ? "multi" : "single"}
        />
      )}
    </SegmentedGroup>
  );

  const nameControl = canCollect ? (
    <SegmentedGroup className={nameClass}>
      <input
        type="text"
        value={attr.name}
        onChange={(e) => updateAttribute(attr.id, "name", e.target.value)}
        placeholder={t("stepEditor.attributeNamePlaceholder")}
        className={formStyles.argNameInputInline}
      />
      <IconDropdown
        ariaLabel={t("stepEditor.collectSelect")}
        faceIcon={
          <span
            className={formStyles.collectIcon}
            style={{
              maskImage: `url(/icons/collect-${collect}.svg)`,
              WebkitMaskImage: `url(/icons/collect-${collect}.svg)`,
            }}
          />
        }
        onChange={(v) => updateAttribute(attr.id, "collect", v)}
        options={collectOptions}
        value={collect}
      />
    </SegmentedGroup>
  ) : (
    <input
      type="text"
      value={attr.name}
      onChange={(e) => updateAttribute(attr.id, "name", e.target.value)}
      placeholder={t("stepEditor.attributeNamePlaceholder")}
      className={`${formStyles.argInput} ${nameClass}`}
    />
  );

  return (
    <div className={formStyles.attrRow}>
      <div className={formStyles.attrRowInputs}>
        <SegmentedGroup className={formStyles.iconDropdownGroup}>
          <IconDropdown
            ariaLabel={t("stepEditor.attrTypeSelect")}
            faceIcon={
              <RoleIcon className={`${styles.iconMd} ${roleClassName}`} />
            }
            onChange={(v) => updateAttribute(attr.id, "role", v)}
            options={roleOptions}
            value={attr.role}
          />
        </SegmentedGroup>
        {typeControl}
        {nameControl}
        {attr.role === "meta" && (
          <ComboInput
            value={attr.metaKey || ""}
            suggestions={META_KEYS}
            onChange={(v) => updateAttribute(attr.id, "metaKey", v)}
            placeholder={t("stepEditor.metaKeyPlaceholder")}
            className={formStyles.argValueInput}
          />
        )}
        {attr.role === "required" && (
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
        {attr.role !== "const" && attr.role !== "meta" && (
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
      {isMappingExpanded && attr.role !== "const" && attr.role !== "meta" && (
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
