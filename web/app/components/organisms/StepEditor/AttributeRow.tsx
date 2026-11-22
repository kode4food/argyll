import React from "react";
import type { Handling, StepType } from "@/app/api";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import { useT } from "@/app/i18n";
import {
  IconCompensate,
  IconExpandDown,
  IconExpandUp,
  IconRemove,
} from "@/utils/iconRegistry";
import type { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import type { Attribute } from "./stepEditorUtils";
import AttributeNameControl from "@/app/components/molecules/AttributeNameControl";
import InlineSelectDropdown from "@/app/components/molecules/InlineSelectDropdown";
import AttributeMappingPanel from "./AttributeMappingPanel";
import AttributeValueControls from "./AttributeValueControls";
import { useAttributeRow } from "./useAttributeRow";

interface AttributeRowProps {
  attr: Attribute;
  stepType: StepType;
  handling: Handling;
  compensatedInnerNames: Set<string>;
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
  handling,
  compensatedInnerNames,
  flowInputOptions,
  flowOutputOptions,
  usedInputMappings,
  usedOutputMappings,
  updateAttribute,
  removeAttribute,
}) => {
  const t = useT();
  const {
    isMappingExpanded,
    toggleMappingExpanded,
    hasMapping,
    hasMappingConfigured,
    isCompensable,
    innerName,
    hasCompensationConflict,
    roleOptions,
    RoleIcon,
    roleClassName,
    canCollect,
    nameClass,
    nameControlClass,
  } = useAttributeRow({ attr, handling, compensatedInnerNames });

  return (
    <div className={formStyles.attrRow}>
      <div className={formStyles.attrRowInputs}>
        <SegmentedGroup className={formStyles.iconDropdownGroup}>
          <InlineSelectDropdown
            ariaLabel={t("stepEditor.attrRoleSelect")}
            faceIcon={
              <RoleIcon className={`${styles.iconMd} ${roleClassName}`} />
            }
            onChange={(v) => updateAttribute(attr.id, "role", v)}
            options={roleOptions}
            value={attr.role}
          />
          {isCompensable && (
            <button
              type="button"
              onClick={() =>
                updateAttribute(attr.id, "compensated", !attr.compensated)
              }
              className={`${formStyles.iconDropdownFace} ${formStyles.compensateButton} ${
                attr.compensated ? formStyles.compensateButtonActive : ""
              }`}
              disabled={hasCompensationConflict}
              aria-pressed={Boolean(attr.compensated)}
              aria-label={`${t("stepEditor.compensatedAttributeLabel")} ${
                attr.name || attr.id
              }`}
              title={
                hasCompensationConflict
                  ? t("stepEditor.compensatedAttributeConflict", {
                      name: innerName,
                    })
                  : t("stepEditor.compensatedAttributeTitle")
              }
            >
              <IconCompensate className={styles.iconSm} />
            </button>
          )}
        </SegmentedGroup>
        <AttributeNameControl
          name={attr.name}
          className={nameControlClass}
          canCollect={canCollect}
          collect={attr.collect}
          forEach={attr.forEach}
          onNameChange={(name) => updateAttribute(attr.id, "name", name)}
          onCollectChange={(collect) =>
            updateAttribute(attr.id, "collect", collect)
          }
          onForEachChange={(forEach) =>
            updateAttribute(attr.id, "forEach", forEach)
          }
        />
        <AttributeValueControls attr={attr} updateAttribute={updateAttribute} />
        {hasMapping && (
          <button
            type="button"
            onClick={toggleMappingExpanded}
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
      {isMappingExpanded && hasMapping && (
        <AttributeMappingPanel
          attr={attr}
          nameClass={nameClass}
          isCompensable={isCompensable}
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
