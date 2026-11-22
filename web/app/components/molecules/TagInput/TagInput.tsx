import React from "react";
import "@yaireo/tagify/dist/tagify.css";
import { type LucideIcon } from "@/utils/iconRegistry";
import styles from "./TagInput.module.css";
import { useTagify } from "./useTagify";

export interface TagInputProps {
  // Rejects any tag that is not one of the suggestions
  shouldEnforceSuggestions?: boolean;
  Icon: LucideIcon;
  // Omitted when the caller heads a group of inputs with a label of its own
  label?: string;
  onChange: (tags: string[]) => void;
  placeholder: string;
  readOnly?: boolean;
  removeLabel: string;
  shouldFocus?: boolean;
  shouldShowFieldIcon?: boolean;
  suggestions: readonly string[];
  tags: string[];
}

const TagInput: React.FC<TagInputProps> = ({
  Icon,
  label,
  shouldShowFieldIcon,
  ...tagifyOptions
}) => {
  const { placeholder, readOnly = false, tags } = tagifyOptions;
  const inputRef = useTagify(tagifyOptions);

  return (
    // Unlabelled inputs are rows inside someone else's section, so they skip
    // the section chrome rather than nesting a second bordered box in it
    <div className={label ? styles.section : undefined}>
      {label && (
        <div className={styles.sectionHeader}>
          <label className={styles.label}>
            <span className={styles.labelIcon}>
              <Icon aria-hidden="true" />
            </span>
            {label}
          </label>
        </div>
      )}
      <div
        className={[styles.field, !label && styles.fieldInline]
          .filter(Boolean)
          .join(" ")}
      >
        {!label && shouldShowFieldIcon && (
          <span className={styles.fieldIcon}>
            <Icon aria-hidden="true" />
          </span>
        )}
        <input
          ref={inputRef}
          aria-label={placeholder}
          readOnly={readOnly}
          defaultValue={tags.join(",")}
        />
      </div>
    </div>
  );
};

export default TagInput;
