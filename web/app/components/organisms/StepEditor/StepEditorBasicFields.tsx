import React from "react";
import type { Handling, StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import useDropdown from "@/app/hooks/useDropdown";
import {
  getStepTypeIcon,
  IconCompensate,
  IconMemoized,
  IconStandard,
  type LucideIcon,
} from "@/utils/iconRegistry";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorBasicFields.module.css";

interface StepEditorBasicFieldsProps {
  handling: Handling;
  isCreateMode: boolean;
  name: string;
  setHandling: (value: Handling) => void;
  setName: (value: string) => void;
  setStepId: (value: string) => void;
  setStepType: (value: StepType) => void;
  stepId: string;
  stepType: StepType;
}

interface FieldSelectOption {
  disabled?: boolean;
  Icon: LucideIcon;
  label: string;
  title?: string;
  value: string;
}

interface FieldSelectProps {
  label: string;
  onChange: (value: string) => void;
  options: FieldSelectOption[];
  value: string;
}

const STEP_TYPE_OPTIONS = [
  {
    value: "sync" as StepType,
    labelKey: "stepEditor.typeSyncLabel",
    titleKey: "stepEditor.typeSyncTitle",
  },
  {
    value: "async" as StepType,
    labelKey: "stepEditor.typeAsyncLabel",
    titleKey: "stepEditor.typeAsyncTitle",
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

const FieldSelect: React.FC<FieldSelectProps> = ({
  label,
  onChange,
  options,
  value,
}) => {
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown(options, value, onChange);
  const selected = options.find((option) => option.value === value);
  const SelectedIcon = selected?.Icon;

  return (
    <div className={`${formStyles.field} ${formStyles.flex1}`}>
      <label className={formStyles.label}>{label}</label>
      <div
        ref={wrapperRef}
        className={localStyles.fieldSelectWrapper}
        onKeyDown={handleKeyDown}
      >
        <button
          type="button"
          onClick={() => setOpen((isOpen) => !isOpen)}
          className={`${localStyles.fieldSelectFace} ${
            open ? localStyles.fieldSelectFaceOpen : ""
          }`}
          aria-expanded={open}
          aria-haspopup="listbox"
        >
          {SelectedIcon && <SelectedIcon className={styles.iconSm} />}
          <span>{selected?.label ?? value}</span>
        </button>
        {open && (
          <div
            className={`${dropdownStyles.list} ${localStyles.fieldSelectList}`}
            role="listbox"
            data-ui-overlay="dropdown"
          >
            {options.map((option, index) => (
              <button
                key={option.value}
                type="button"
                role="option"
                aria-selected={value === option.value}
                title={option.title ?? option.label}
                disabled={option.disabled}
                className={`${dropdownStyles.item} ${
                  value === option.value ? dropdownStyles.itemActive : ""
                } ${option.disabled ? dropdownStyles.itemDisabled : ""} ${
                  index === highlightedIndex
                    ? dropdownStyles.itemHighlighted
                    : ""
                }`}
                onMouseEnter={() => setHighlightedIndex(index)}
                onClick={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
              >
                <span className={dropdownStyles.itemIcon}>
                  <option.Icon className={styles.iconSm} />
                </span>
                <span className={dropdownStyles.itemLabel}>{option.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const StepEditorBasicFields: React.FC<StepEditorBasicFieldsProps> = ({
  handling,
  isCreateMode,
  name,
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
  const handlingOptions: FieldSelectOption[] = [
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
      disabled: stepType !== "sync" && stepType !== "async",
      Icon: IconCompensate,
      label: t("stepEditor.handling.compensated"),
      value: "compensated",
    },
  ];

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
      <FieldSelect
        label={t("stepEditor.typeLabel")}
        onChange={(value) => setStepType(value as StepType)}
        options={typeOptions}
        value={stepType}
      />
      <FieldSelect
        label={t("stepEditor.handlingLabel")}
        onChange={(value) => setHandling(value as Handling)}
        options={handlingOptions}
        value={handling}
      />
    </div>
  );
};

export default StepEditorBasicFields;
