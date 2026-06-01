import React from "react";
import { StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import useDropdown from "@/app/hooks/useDropdown";
import { getStepTypeIcon } from "@/utils/iconRegistry";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
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
    value: "sync",
    labelKey: "stepEditor.typeSyncLabel",
    titleKey: "stepEditor.typeSyncTitle",
  },
  {
    type: "async" as StepType,
    value: "async",
    labelKey: "stepEditor.typeAsyncLabel",
    titleKey: "stepEditor.typeAsyncTitle",
  },
  {
    type: "script" as StepType,
    value: "script",
    labelKey: "stepEditor.typeScriptLabel",
    titleKey: "stepEditor.typeScriptTitle",
  },
  {
    type: "flow" as StepType,
    value: "flow",
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

  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown(STEP_TYPE_OPTIONS, stepType, (value) =>
    setStepType(value as StepType)
  );

  const selectedOption = STEP_TYPE_OPTIONS.find((o) => o.type === stepType);
  const SelectedIcon = getStepTypeIcon(stepType);

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
        <div
          ref={wrapperRef}
          className={localStyles.typeSelectWrapper}
          onKeyDown={handleKeyDown}
        >
          <button
            type="button"
            onClick={() => setOpen((o) => !o)}
            className={`${localStyles.typeSelectFace} ${open ? localStyles.typeSelectFaceOpen : ""}`}
            aria-expanded={open}
            aria-haspopup="listbox"
          >
            <SelectedIcon className={styles.iconSm} />
            <span>
              {selectedOption ? t(selectedOption.labelKey) : stepType}
            </span>
          </button>
          {open && (
            <div
              className={`${dropdownStyles.list} ${localStyles.typeSelectList}`}
              role="listbox"
              data-ui-overlay="dropdown"
            >
              {STEP_TYPE_OPTIONS.map(({ type, labelKey, titleKey }, index) => {
                const Icon = getStepTypeIcon(type);
                return (
                  <button
                    key={type}
                    type="button"
                    role="option"
                    aria-selected={stepType === type}
                    title={t(titleKey)}
                    className={`${dropdownStyles.item} ${stepType === type ? dropdownStyles.itemActive : ""} ${index === highlightedIndex ? dropdownStyles.itemHighlighted : ""}`}
                    onMouseEnter={() => setHighlightedIndex(index)}
                    onClick={() => {
                      setStepType(type);
                      setOpen(false);
                    }}
                  >
                    <span className={dropdownStyles.itemIcon}>
                      <Icon className={styles.iconSm} />
                    </span>
                    <span className={dropdownStyles.itemLabel}>
                      {t(labelKey)}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default StepEditorBasicFields;
