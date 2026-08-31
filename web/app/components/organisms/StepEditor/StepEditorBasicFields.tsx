import React from "react";
import type { Handling, StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import SelectField, {
  type SelectFieldOption,
} from "@/app/components/molecules/SelectField";
import {
  getStepTypeIcon,
  IconCompensate,
  IconMemoized,
  IconStandard,
} from "@/utils/iconRegistry";
import formStyles from "./StepEditorForm.module.css";

interface StepEditorBasicFieldsProps {
  description: string;
  handling: Handling;
  isCreateMode: boolean;
  name: string;
  setDescription: (value: string) => void;
  setHandling: (value: Handling) => void;
  setName: (value: string) => void;
  setStepId: (value: string) => void;
  setStepType: (value: StepType) => void;
  stepId: string;
  stepType: StepType;
}

const STEP_TYPE_OPTIONS = [
  {
    value: "service" as StepType,
    labelKey: "stepEditor.typeServiceLabel",
    titleKey: "stepEditor.typeServiceTitle",
  },
  {
    value: "script" as StepType,
    labelKey: "stepEditor.typeScriptLabel",
    titleKey: "stepEditor.typeScriptTitle",
  },
  {
    value: "flow" as StepType,
    labelKey: "stepEditor.typeFlowLabel",
    titleKey: "stepEditor.typeFlowTitle",
  },
];

const StepEditorBasicFields: React.FC<StepEditorBasicFieldsProps> = ({
  description,
  handling,
  isCreateMode,
  name,
  setDescription,
  setHandling,
  setName,
  setStepId,
  setStepType,
  stepId,
  stepType,
}) => {
  const t = useT();
  const typeOptions = STEP_TYPE_OPTIONS.map(
    ({ value, labelKey, titleKey }) => ({
      Icon: getStepTypeIcon(value),
      label: t(labelKey),
      title: t(titleKey),
      value,
    })
  );
  const handlingOptions: SelectFieldOption[] = [
    {
      Icon: IconStandard,
      label: t("stepEditor.handling.standard"),
      value: "standard",
    },
    {
      Icon: IconMemoized,
      label: t("stepEditor.handling.memoized"),
      value: "memoized",
    },
    {
      disabled: stepType !== "service",
      Icon: IconCompensate,
      label: t("stepEditor.handling.compensated"),
      value: "compensated",
    },
  ];

  return (
    <>
      <div className={formStyles.row}>
        <div className={`${formStyles.field} ${formStyles.flex1}`}>
          <label className={formStyles.label}>
            {t("stepEditor.stepIdLabel")}
          </label>
          <input
            type="text"
            value={stepId}
            onChange={(e) => setStepId(e.target.value)}
            className={formStyles.formControl}
            disabled={!isCreateMode}
            placeholder={t("stepEditor.stepIdPlaceholder")}
          />
        </div>
        <div className={`${formStyles.field} ${formStyles.flex2}`}>
          <label className={formStyles.label}>
            {t("stepEditor.stepNameLabel")}
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={formStyles.formControl}
            placeholder={t("stepEditor.stepNamePlaceholder")}
          />
        </div>
        <SelectField
          className={formStyles.flex1}
          label={t("stepEditor.typeLabel")}
          onChange={(value) => setStepType(value as StepType)}
          options={typeOptions}
          value={stepType}
        />
        <SelectField
          className={formStyles.flex1}
          label={t("stepEditor.handlingLabel")}
          onChange={(value) => setHandling(value as Handling)}
          options={handlingOptions}
          value={handling}
        />
      </div>
      <div className={formStyles.row}>
        <div className={`${formStyles.field} ${formStyles.flex1}`}>
          <label className={formStyles.label}>
            {t("stepEditor.descriptionLabel")}
          </label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className={formStyles.formControl}
            placeholder={t("stepEditor.descriptionPlaceholder")}
          />
        </div>
      </div>
    </>
  );
};

export default StepEditorBasicFields;
