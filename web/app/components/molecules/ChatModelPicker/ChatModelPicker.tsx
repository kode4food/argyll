import React from "react";
import SelectField, {
  textOptions,
} from "@/app/components/molecules/SelectField";
import styles from "./ChatModelPicker.module.css";

export interface ChatModelPickerProps {
  providers: string[];
  models: string[];
  provider: string;
  model: string;
  providerLabel: string;
  modelLabel: string;
  onProviderChange: (provider: string) => void;
  onModelChange: (model: string) => void;
}

const ChatModelPicker: React.FC<ChatModelPickerProps> = ({
  providers,
  models,
  provider,
  model,
  providerLabel,
  modelLabel,
  onProviderChange,
  onModelChange,
}) => (
  <div className={styles.picker}>
    <SelectField
      className={styles.provider}
      ariaLabel={providerLabel}
      value={provider}
      onChange={onProviderChange}
      options={textOptions(providers)}
      placeholder={providerLabel}
    />
    <SelectField
      className={styles.model}
      ariaLabel={modelLabel}
      value={model}
      onChange={onModelChange}
      options={textOptions(models)}
      placeholder={modelLabel}
    />
  </div>
);

export default ChatModelPicker;
