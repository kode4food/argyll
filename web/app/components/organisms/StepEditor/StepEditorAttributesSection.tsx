import React from "react";
import { StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import { IconAdd } from "@/utils/iconRegistry";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { Attribute } from "./stepEditorUtils";
import AttributeRow from "./AttributeRow";

interface StepEditorAttributesSectionProps {
  addAttribute: () => void;
  attributes: Attribute[];
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
  flowInputOptions,
  flowOutputOptions,
  removeAttribute,
  stepType,
  updateAttribute,
}) => {
  const t = useT();

  const usedInputMappings = new Map<string, string>();
  const usedOutputMappings = new Map<string, string>();
  attributes.forEach((attr) => {
    const mappingName = attr.mappingName?.trim();
    if (!mappingName) return;
    if (attr.role === "output") {
      usedOutputMappings.set(mappingName, attr.id);
    } else {
      usedInputMappings.set(mappingName, attr.id);
    }
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
        {attributes.map((attr) => (
          <AttributeRow
            key={attr.id}
            attr={attr}
            stepType={stepType}
            flowInputOptions={flowInputOptions}
            flowOutputOptions={flowOutputOptions}
            usedInputMappings={usedInputMappings}
            usedOutputMappings={usedOutputMappings}
            updateAttribute={updateAttribute}
            removeAttribute={removeAttribute}
          />
        ))}
      </div>
    </div>
  );
};

export default StepEditorAttributesSection;
