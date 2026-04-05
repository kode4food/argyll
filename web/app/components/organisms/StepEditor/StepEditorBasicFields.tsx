import React from "react";
import { StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import { getStepTypeIcon } from "@/utils/iconRegistry";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorBasicFields.module.css";

interface StepEditorBasicFieldsProps {
  isCreateMode: boolean;
  name: string;
  setName: (value: string) => void;
  setStepId: (value: string) => void;
  setStepType: (value: StepType) => void;
  stepId: string;
  stepType: StepType;
}

const STEP_TYPE_OPTIONS = [
  {
    type: "sync" as StepType,
    labelKey: "stepEditor.typeSyncLabel",
    titleKey: "stepEditor.typeSyncTitle",
  },
  {
    type: "async" as StepType,
    labelKey: "stepEditor.typeAsyncLabel",
    titleKey: "stepEditor.typeAsyncTitle",
  },
  {
    type: "script" as StepType,
    labelKey: "stepEditor.typeScriptLabel",
    titleKey: "stepEditor.typeScriptTitle",
  },
  {
    type: "flow" as StepType,
    labelKey: "stepEditor.typeFlowLabel",
    titleKey: "stepEditor.typeFlowTitle",
  },
];

const StepEditorBasicFields: React.FC<StepEditorBasicFieldsProps> = ({
  isCreateMode,
  name,
  setName,
  setStepId,
  setStepType,
  stepId,
  stepType,
}) => {
  const t = useT();

  return (
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
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.label}>{t("stepEditor.typeLabel")}</label>
        <div className={localStyles.typeButtonGroup}>
          {STEP_TYPE_OPTIONS.map(({ type, labelKey, titleKey }) => {
            const Icon = getStepTypeIcon(type);
            return (
              <button
                key={type}
                type="button"
                onClick={(e) => {
                  setStepType(type);
                  e.currentTarget.blur();
                }}
                className={`${localStyles.typeButton} ${stepType === type ? localStyles.typeButtonActive : ""}`}
                title={t(titleKey)}
              >
                <Icon className={styles.iconSm} />
                <span>{t(labelKey)}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default StepEditorBasicFields;
