import React from "react";
import { IconAdd, IconRemove, type LucideIcon } from "@/utils/iconRegistry";
import ComboInput from "@/app/components/molecules/ComboInput";
import styles from "./KeyValueTable.module.css";

export interface KeyValuePair {
  id: string;
  key: string;
  value: string;
}

export interface KeyValueTableProps {
  addLabel: string;
  Icon: LucideIcon;
  keySuggestions: readonly string[];
  keyPlaceholder: string;
  label: string;
  onAdd: () => void;
  onChange: (id: string, field: "key" | "value", value: string) => void;
  onRemove: (id: string) => void;
  pairs: KeyValuePair[];
  removeLabel: string;
  valuePlaceholder: string;
  valueSuggestions: (key: string) => readonly string[];
}

const KeyValueTable: React.FC<KeyValueTableProps> = ({
  addLabel,
  Icon,
  keySuggestions,
  keyPlaceholder,
  label,
  onAdd,
  onChange,
  onRemove,
  pairs,
  removeLabel,
  valuePlaceholder,
  valueSuggestions,
}) => (
  <div className={styles.section}>
    <div className={styles.sectionHeader}>
      <label className={styles.label}>
        <span className={styles.labelIcon}>
          <Icon aria-hidden="true" />
        </span>
        {label}
      </label>
      <button
        type="button"
        onClick={onAdd}
        className={styles.addButton}
        title={addLabel}
      >
        <IconAdd className={styles.icon} />
      </button>
    </div>
    <div className={styles.rows}>
      {pairs.map((pair) => (
        <div key={pair.id} className={styles.row}>
          <ComboInput
            className={styles.keyInput}
            value={pair.key}
            onChange={(value) => onChange(pair.id, "key", value)}
            placeholder={keyPlaceholder}
            ariaLabel={keyPlaceholder}
            suggestions={keySuggestions.filter(
              (key) =>
                !pairs.some(
                  (other) => other.id !== pair.id && other.key === key
                )
            )}
          />
          <ComboInput
            className={styles.valueInput}
            value={pair.value}
            onChange={(value) => onChange(pair.id, "value", value)}
            placeholder={valuePlaceholder}
            ariaLabel={valuePlaceholder}
            suggestions={valueSuggestions(pair.key)}
          />
          <button
            type="button"
            onClick={() => onRemove(pair.id)}
            className={styles.removeButton}
            title={removeLabel}
            aria-label={`${removeLabel} ${pair.key || pair.id}`}
          >
            <IconRemove className={styles.iconSm} />
          </button>
        </div>
      ))}
    </div>
  </div>
);

export default KeyValueTable;
