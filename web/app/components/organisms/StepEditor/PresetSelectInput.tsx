import React from "react";
import SelectField from "@/app/components/molecules/SelectField";
import { type LucideIcon } from "@/utils/iconRegistry";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import InlineSelectDropdown, {
  type InlineSelectOption,
} from "./InlineSelectDropdown";

export interface PresetSelectInputProps {
  ariaLabel: string;
  Icon: LucideIcon;
  onChange: (value: string) => void;
  presets: InlineSelectOption[];
  placeholder: string;
  value: string;
  customLabel?: string;
  emptyLabel?: string;
  renderCustom?: (picker: React.ReactNode) => React.ReactNode;
}

export const PRESET_CUSTOM = "custom";

const PresetSelectInput: React.FC<PresetSelectInputProps> = ({
  ariaLabel,
  Icon,
  onChange,
  presets,
  placeholder,
  value,
  customLabel,
  emptyLabel,
  renderCustom,
}) => {
  const custom = customLabel
    ? [{ value: PRESET_CUSTOM, label: customLabel }]
    : [];

  if (renderCustom) {
    return renderCustom(
      <InlineSelectDropdown
        ariaLabel={ariaLabel}
        faceIcon={<Icon className={styles.iconMd} />}
        onChange={onChange}
        options={[...presets, ...custom]}
        value={PRESET_CUSTOM}
      />
    );
  }

  // A chosen preset reads as its label with the raw value beside it
  const labeled = presets.map((preset) => ({
    value: preset.value,
    label: `${preset.label} (${preset.value})`,
  }));
  const empty =
    emptyLabel && presets.length === 0
      ? [{ value: "", label: emptyLabel, disabled: true }]
      : [];

  return (
    <SelectField
      ariaLabel={ariaLabel}
      className={formStyles.argValueInput}
      Icon={Icon}
      onChange={onChange}
      options={[...labeled, ...empty, ...custom]}
      placeholder={placeholder}
      value={value}
    />
  );
};

export default PresetSelectInput;
