import React from "react";
import SelectField from "@/app/components/molecules/SelectField";
import TagInput from "@/app/components/molecules/TagInput";
import ScriptConfigEditor from "@/app/components/organisms/StepEditor/ScriptConfigEditor";
import { useT } from "@/app/i18n";
import { useSpaces, useSteps, useWatchedFeeds } from "@/app/store/flowStore";
import formStyles from "@/app/styles/components/form.module.css";
import styles from "@/app/styles/components/manager.module.css";
import { IconAttributeOutput, IconFeed, IconSpace } from "@/utils/iconRegistry";
import { formatJSON } from "@/utils/jsonUtils";
import { foldCompletions, outputNames } from "./feedManagerUtils";
import { FeedEditor } from "./useFeedEditor";
import feedStyles from "./FeedManager.module.css";

interface FeedManagerFormProps {
  editor: FeedEditor;
}

const FeedManagerForm: React.FC<FeedManagerFormProps> = ({ editor }) => {
  const t = useT();
  const watchedFeeds = useWatchedFeeds();
  const steps = useSteps();
  const spaces = useSpaces();
  const { draft, editingId, patchDraft } = editor;
  const watchedFeed = editingId ? watchedFeeds[editingId] : undefined;
  const isIdLocked = editingId !== null;
  const outputs = React.useMemo(() => outputNames(steps), [steps]);
  const completions = React.useMemo(() => foldCompletions(draft), [draft]);
  const spaceOptions = React.useMemo(
    () => [
      { value: "", label: t("feedManager.spaceAny"), Icon: IconSpace },
      ...spaces.map((s) => ({
        value: s.id,
        label: s.name,
        title: s.description,
        Icon: IconSpace,
      })),
    ],
    [spaces, t]
  );

  return (
    <div className={styles.detail}>
      <div className={feedStyles.row}>
        <div
          className={`${styles.field} ${feedStyles.flex1} ${isIdLocked ? formStyles.disabledField : ""}`}
        >
          <label className={styles.label}>{t("feedManager.idLabel")}</label>
          <input
            type="text"
            className={styles.input}
            value={draft.id}
            disabled={isIdLocked}
            onChange={(e) => patchDraft({ id: e.target.value })}
            placeholder={t("feedManager.idPlaceholder")}
          />
        </div>
        <div className={`${styles.field} ${feedStyles.flex2}`}>
          <label className={styles.label}>{t("feedManager.nameLabel")}</label>
          <input
            type="text"
            className={styles.input}
            value={draft.name}
            onChange={(e) => patchDraft({ name: e.target.value })}
            placeholder={t("feedManager.namePlaceholder")}
          />
        </div>
      </div>

      <div className={styles.field}>
        <label className={styles.label}>
          {t("feedManager.descriptionLabel")}
        </label>
        <input
          type="text"
          className={styles.input}
          value={draft.description}
          onChange={(e) => patchDraft({ description: e.target.value })}
          placeholder={t("feedManager.descriptionPlaceholder")}
        />
      </div>

      <div className={feedStyles.row}>
        <div className={`${styles.field} ${feedStyles.flex1}`}>
          <label className={styles.label}>{t("feedManager.seedLabel")}</label>
          <input
            type="text"
            className={styles.input}
            value={draft.value}
            onChange={(e) => patchDraft({ value: e.target.value })}
            placeholder={t("feedManager.seedPlaceholder")}
          />
        </div>
        <SelectField
          className={feedStyles.flex1}
          label={t("feedManager.spaceLabel")}
          ariaLabel={t("feedManager.spaceLabel")}
          options={spaceOptions}
          value={draft.spaceId}
          onChange={(spaceId) => patchDraft({ spaceId })}
        />
      </div>

      <div className={styles.field}>
        <label className={styles.label}>
          {t("feedManager.attributesLabel")}
        </label>
        <TagInput
          shouldEnforceSuggestions
          Icon={IconAttributeOutput}
          removeLabel={t("feedManager.removeAttribute")}
          shouldShowFieldIcon
          placeholder={t("feedManager.attributesPlaceholder")}
          tags={draft.attributes}
          onChange={(attributes) => patchDraft({ attributes })}
          suggestions={outputs}
        />
      </div>

      <ScriptConfigEditor
        Icon={IconFeed}
        label={t("feedManager.foldLabel")}
        completions={completions}
        value={draft.fold.source}
        onChange={(script) => editor.patchFold({ source: script })}
        language={draft.fold.language}
        onLanguageChange={(language) => editor.patchFold({ language })}
      />
      <div className={styles.note}>{t("feedManager.foldHint")}</div>

      {watchedFeed && (
        <div className={styles.field}>
          <label className={styles.label}>
            {t("feedManager.currentValueLabel")}
          </label>
          <pre className={feedStyles.value}>
            {formatJSON(watchedFeed.value)}
          </pre>
        </div>
      )}
    </div>
  );
};

export default FeedManagerForm;
