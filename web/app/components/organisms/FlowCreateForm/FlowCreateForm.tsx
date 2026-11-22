import React from "react";
import { Step } from "@/app/api";
import { useT } from "@/app/i18n";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import CatalogMenu from "./CatalogMenu";
import FlowGoalsSection from "./FlowGoalsSection";
import SelectField from "@/app/components/molecules/SelectField";
import FlowAttributesSection from "./FlowAttributesSection";
import FlowStartSection from "./FlowStartSection";
import styles from "./FlowCreateForm.module.css";
import { useFlowCreateForm } from "./useFlowCreateForm";

interface FlowCreateFormProps {
  onCreateStep?: (draft?: Step) => void;
}

const getFlowInputPlaceholder = (option: FlowInputOption): string =>
  option.defaultValue ?? "";

const FlowCreateForm: React.FC<FlowCreateFormProps> = ({ onCreateStep }) => {
  const t = useT();
  const {
    newID,
    handleStepChange,
    initialState,
    setInitialState,
    creating,
    handleCreateFlow,
    steps,
    goalSteps,
    setHoveredGoalSet,
    setFocusedPreviewAttribute,
    spaceId,
    jsonError,
    spaceOptions,
    scopedSteps,
    sortedSteps,
    sidebarFade,
    stepFiltering,
    flowInputOptions,
    emptyAttributesLabel,
    inputDrafts,
    isStartSectionDisabled,
    handleCreateStep,
    handleSpaceChange,
    handleFlowIdChange,
    handleGenerateId,
  } = useFlowCreateForm(onCreateStep);

  return (
    <div className={styles.panel}>
      <div className={styles.container}>
        <div className={styles.main}>
          <div className={styles.panelBody}>
            <div className={styles.spaceRow}>
              <SelectField
                className={styles.spaceSelect}
                ariaLabel={t("flowCreate.spaceLabel")}
                onChange={handleSpaceChange}
                options={spaceOptions}
                value={spaceId ?? ""}
              />
              <CatalogMenu />
            </div>

            <FlowGoalsSection
              goalSteps={goalSteps}
              onPreviewGoalSet={setHoveredGoalSet}
              blockedByStep={stepFiltering.blockedByStep}
              missingByStep={stepFiltering.missingByStep}
              onCreateStep={handleCreateStep}
              onGoalStepsChange={handleStepChange}
              satisfied={stepFiltering.satisfied}
              showBottomFade={sidebarFade.showBottomFade}
              showTopFade={sidebarFade.showTopFade}
              sidebarListRef={sidebarFade.scrollRef}
              sortedSteps={sortedSteps}
              spaceScoped={!!spaceId}
              stepsCount={scopedSteps.length}
            />

            <FlowAttributesSection
              editorMode={inputDrafts.editorMode}
              emptyAttributesLabel={emptyAttributesLabel}
              flowInputOptions={flowInputOptions}
              flowInputValues={inputDrafts.flowInputValues}
              flowInputValuesRaw={inputDrafts.flowInputValuesRaw}
              getFlowInputPlaceholder={getFlowInputPlaceholder}
              handleBasicInputBlur={inputDrafts.commitBasicInputValues}
              handleBasicInputChange={inputDrafts.handleBasicInputChange}
              initialState={initialState}
              jsonError={jsonError}
              onEditorModeChange={inputDrafts.handleEditorModeChange}
              onFocusedPreviewAttributeChange={setFocusedPreviewAttribute}
              setInitialState={setInitialState}
            />
          </div>

          <div className={styles.panelFooter}>
            <FlowStartSection
              creating={creating}
              disabled={isStartSectionDisabled}
              flowId={newID}
              onCreateFlow={handleCreateFlow}
              onFlowIdChange={handleFlowIdChange}
              onGenerateId={handleGenerateId}
            />
          </div>
        </div>
      </div>

      {steps.length === 0 && (
        <div className={styles.warning}>{t("flowCreate.warningNoSteps")}</div>
      )}
    </div>
  );
};

export default FlowCreateForm;
