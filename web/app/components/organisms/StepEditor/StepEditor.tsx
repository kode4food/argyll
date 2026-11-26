import React from "react";
import Modal from "@/app/components/molecules/Modal";
import { Step } from "@/app/api";
import ScriptConfigEditor from "./ScriptConfigEditor";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import fadeStyles from "@/app/styles/components/scrollFade.module.css";
import { useT } from "@/app/i18n";
import { IconPredicate } from "@/utils/iconRegistry";
import StepEditorBasicFields from "./StepEditorBasicFields";
import StepEditorAttributesSection from "./StepEditorAttributesSection";
import StepEditorTypeConfiguration from "./StepEditorTypeConfiguration";
import StepEditorFooter from "./StepEditorFooter";
import { useStepEditor } from "./useStepEditor";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  draft?: Step;
}

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
  draft,
}) => {
  const t = useT();
  const {
    form,
    steps,
    tagVocabulary,
    editorMode,
    jsonDraft,
    setJsonDraft,
    predicateCompletions,
    flowPreview,
    flowInputOptions,
    flowOutputOptions,
    dimensions,
    mounted,
    scrollRef,
    showTopFade,
    showBottomFade,
    handleEditorModeChange,
    handleSaveClick,
    isDirty,
    title,
  } = useStepEditor({ step, onClose, onUpdate, diagramContainerRef, draft });

  if (!mounted) return null;

  return (
    <Modal
      isOpen
      onClose={onClose}
      width={dimensions.width}
      height={dimensions.height}
      title={title}
      footer={
        <StepEditorFooter
          editorMode={editorMode}
          onEditorModeChange={handleEditorModeChange}
          onCancel={onClose}
          onSave={handleSaveClick}
          saving={form.saving}
          isCreateMode={form.isCreateMode}
          isDirty={isDirty}
        />
      }
    >
      <>
        <div
          ref={scrollRef}
          className={`${formStyles.formContainer} ${
            editorMode === "json" ? formStyles.formContainerJsonMode : ""
          } ${showTopFade ? fadeStyles.fadeTop : ""} ${
            showBottomFade ? fadeStyles.fadeBottom : ""
          }`}
        >
          {editorMode === "basic" ? (
            <>
              <StepEditorBasicFields
                description={form.description}
                handling={form.handling}
                isCreateMode={form.isCreateMode}
                name={form.name}
                setDescription={form.setDescription}
                setHandling={form.setHandling}
                setName={form.setName}
                setStepId={form.setStepId}
                setStepType={form.setStepType}
                setTags={form.setTags}
                stepId={form.stepId}
                stepType={form.stepType}
                tags={form.tags}
                tagVocabulary={tagVocabulary}
              />

              <StepEditorAttributesSection
                addAttribute={form.addAttribute}
                attributes={form.attributes}
                flowInputOptions={flowInputOptions}
                flowOutputOptions={flowOutputOptions}
                removeAttribute={form.removeAttribute}
                stepType={form.stepType}
                handling={form.handling}
                updateAttribute={form.updateAttribute}
              />

              <ScriptConfigEditor
                Icon={IconPredicate}
                label={t("stepEditor.predicateLabel")}
                completions={predicateCompletions}
                value={form.predicate}
                onChange={form.setPredicate}
                language={form.predicateLanguage}
                onLanguageChange={form.setPredicateLanguage}
                containerClassName={formStyles.predicateEditorContainer}
              />

              <StepEditorTypeConfiguration
                form={form}
                steps={steps}
                flowPreview={flowPreview}
              />
            </>
          ) : (
            <div className={formStyles.jsonSection}>
              <div className={formStyles.jsonEditorContainer}>
                <ScriptEditor
                  value={jsonDraft}
                  onChange={setJsonDraft}
                  language="json"
                />
              </div>
            </div>
          )}
        </div>

        {form.error && (
          <div className={`${formStyles.errorMessage} ${styles.errorBanner}`}>
            {form.error}
          </div>
        )}
      </>
    </Modal>
  );
};

export default StepEditor;
