import React from "react";
import TagInput from "@/app/components/molecules/TagInput";
import { useT } from "@/app/i18n";
import { IconAdd, IconAttributeLabel, IconRemove } from "@/utils/iconRegistry";
import styles from "./SpaceManager.module.css";

interface SpaceManagerSelectorStepProps {
  isAddDisabled: boolean;
  isEditScriptDisabled: boolean;
  onEditScript: () => void;
  onTermsChange: (qbe: string[][]) => void;
  suggestions: readonly string[];
  terms: string[][];
}

const SpaceManagerSelectorStep: React.FC<SpaceManagerSelectorStepProps> = ({
  isAddDisabled,
  isEditScriptDisabled,
  onEditScript,
  onTermsChange,
  suggestions,
  terms,
}) => {
  const t = useT();
  const [focusTermIndex, setFocusTermIndex] = React.useState(-1);
  const addAlternative = t("spaceManager.addAlternative");
  const removeAlternative = t("spaceManager.removeAlternative");
  return (
    <div className={styles.detail}>
      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <label className={styles.labelWithIcon}>
            <span className={styles.labelIcon}>
              <IconAttributeLabel aria-hidden="true" />
            </span>
            {t("spaceManager.selectorLabel")}
          </label>
          <button
            type="button"
            className={styles.addTerm}
            title={addAlternative}
            aria-label={addAlternative}
            disabled={isAddDisabled}
            onClick={() => {
              setFocusTermIndex(terms.length);
              onTermsChange([...terms, []]);
            }}
          >
            <IconAdd className={styles.iconSm} />
          </button>
        </div>
        {terms.map((term, index) => (
          <div key={index} className={styles.term}>
            <TagInput
              Icon={IconAttributeLabel}
              removeLabel={t("spaceManager.removeSelector")}
              placeholder={t("stepEditor.tagPlaceholder")}
              shouldFocus={
                index === focusTermIndex || (terms.length === 1 && !term.length)
              }
              tags={term}
              onChange={(tags) =>
                onTermsChange(
                  terms.map((current, at) => (at === index ? tags : current))
                )
              }
              suggestions={suggestions}
            />
            {index > 0 && (
              <span className={styles.or}>{t("spaceManager.or")}</span>
            )}
            {terms.length > 1 && (
              <button
                type="button"
                className={styles.removeTerm}
                title={removeAlternative}
                aria-label={`${removeAlternative} ${index + 1}`}
                onClick={() =>
                  onTermsChange(terms.filter((_, at) => at !== index))
                }
              >
                <IconRemove className={styles.iconSm} />
              </button>
            )}
          </div>
        ))}
      </div>
      <button
        type="button"
        className={styles.advanced}
        onClick={onEditScript}
        disabled={isEditScriptDisabled}
      >
        {t("spaceManager.editScript")}
      </button>
    </div>
  );
};

export default SpaceManagerSelectorStep;
