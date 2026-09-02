import React from "react";
import { useT } from "@/app/i18n";
import formStyles from "@/app/styles/components/form.module.css";
import { SpaceDraft } from "./spaceManagerUtils";
import styles from "./SpaceManager.module.css";

interface SpaceManagerDetailsStepProps {
  draft: SpaceDraft;
  isIdLocked: boolean;
  onPatch: (patch: Partial<SpaceDraft>) => void;
  selectorDescription: string;
}

const SpaceManagerDetailsStep: React.FC<SpaceManagerDetailsStepProps> = ({
  draft,
  isIdLocked,
  onPatch,
  selectorDescription,
}) => {
  const t = useT();
  return (
    <div className={styles.detail}>
      <div
        className={`${styles.field} ${isIdLocked ? formStyles.disabledField : ""}`}
      >
        <label className={styles.label}>{t("spaceManager.idLabel")}</label>
        <input
          type="text"
          className={styles.input}
          value={draft.id}
          disabled={isIdLocked}
          onChange={(e) => onPatch({ id: e.target.value })}
          placeholder={t("spaceManager.idPlaceholder")}
        />
      </div>

      <div className={styles.field}>
        <label className={styles.label}>{t("spaceManager.nameLabel")}</label>
        <input
          type="text"
          className={styles.input}
          value={draft.name}
          onChange={(e) => onPatch({ name: e.target.value })}
          placeholder={t("spaceManager.namePlaceholder")}
        />
      </div>

      <div className={styles.field}>
        <label className={styles.label}>
          {t("spaceManager.descriptionLabel")}
        </label>
        <input
          type="text"
          className={styles.input}
          value={draft.description}
          onChange={(e) => onPatch({ description: e.target.value })}
          placeholder={
            selectorDescription
              ? t("spaceManager.descriptionSuggestion", {
                  selector: selectorDescription,
                })
              : t("spaceManager.descriptionPlaceholder")
          }
        />
      </div>
    </div>
  );
};

export default SpaceManagerDetailsStep;
