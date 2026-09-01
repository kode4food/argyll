import React from "react";
import TagInput from "@/app/components/molecules/TagInput";
import { useT } from "@/app/i18n";
import { IconAttributeLabel } from "@/utils/iconRegistry";
import styles from "./SpaceManager.module.css";

interface SpaceManagerSelectorStepProps {
  isEditScriptDisabled: boolean;
  onEditScript: () => void;
  onTagsChange: (qbe: string[]) => void;
  suggestions: readonly string[];
  tags: string[];
}

const SpaceManagerSelectorStep: React.FC<SpaceManagerSelectorStepProps> = ({
  isEditScriptDisabled,
  onEditScript,
  onTagsChange,
  suggestions,
  tags,
}) => {
  const t = useT();
  return (
    <div className={styles.detail}>
      <TagInput
        Icon={IconAttributeLabel}
        label={t("spaceManager.selectorLabel")}
        removeLabel={t("spaceManager.removeSelector")}
        placeholder={t("stepEditor.tagPlaceholder")}
        tags={tags}
        onChange={onTagsChange}
        suggestions={suggestions}
      />
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
