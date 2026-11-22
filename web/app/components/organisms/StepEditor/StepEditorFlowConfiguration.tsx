import React from "react";
import { useT } from "@/app/i18n";
import {
  IconClose,
  IconFlowGoals,
  IconGoalFallback,
  IconSpace,
} from "@/utils/iconRegistry";
import SelectField from "@/app/components/molecules/SelectField";
import GoalSetLine from "./GoalSetLine";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorFlowConfiguration.module.css";
import {
  StepEditorFlowConfigurationOptions,
  useStepEditorFlowConfiguration,
} from "./useStepEditorFlowConfiguration";

const StepEditorFlowConfiguration: React.FC<
  StepEditorFlowConfigurationOptions
> = (props) => {
  const t = useT();
  const {
    spaceOptions,
    displaySteps,
    context,
    committedLines,
    activeLine,
    activeLineIdx,
    handleLineChange,
    handleSpaceChange,
    handleRemoveSet,
    handleAddFallback,
  } = useStepEditorFlowConfiguration(props);

  return (
    <div className={formStyles.section}>
      <div className={localStyles.flowGoalRow}>
        <div className={localStyles.flowSpaceColumn}>
          <div className={formStyles.sectionHeader}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconSpace aria-hidden="true" />
              </span>
              {t("flowCreate.spaceLabel")}
            </label>
          </div>
          <SelectField
            ariaLabel={t("flowCreate.spaceLabel")}
            onChange={handleSpaceChange}
            options={spaceOptions}
            value={props.flowSpaceId}
          />
        </div>
        <div className={localStyles.flowGoalColumn}>
          <div className={formStyles.sectionHeader}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconFlowGoals aria-hidden="true" />
              </span>
              {t("stepEditor.flowGoalsLabel")}
            </label>
          </div>
          {committedLines.length > 0 && (
            <div className={localStyles.flowGoalSets}>
              {committedLines.map((set, idx) => (
                <React.Fragment key={idx}>
                  <div className={localStyles.flowGoalGroup}>
                    <GoalSetLine
                      context={context}
                      onChange={(next) => handleLineChange(idx, next)}
                      set={set}
                      steps={displaySteps}
                      t={t}
                    />
                    <button
                      type="button"
                      className={localStyles.flowGoalSetRemove}
                      aria-label={t("goals.removeSet")}
                      onClick={() => handleRemoveSet(idx)}
                    >
                      <IconClose aria-hidden="true" />
                    </button>
                  </div>
                  <span className={localStyles.flowGoalOr}>
                    {t("goals.or")}
                  </span>
                </React.Fragment>
              ))}
            </div>
          )}
          <GoalSetLine
            context={context}
            onChange={(next) => handleLineChange(activeLineIdx, next)}
            set={activeLine}
            steps={displaySteps}
            t={t}
          />
          <button
            type="button"
            className={localStyles.flowGoalOrButton}
            disabled={activeLine.length === 0}
            aria-label={t("goals.addFallback")}
            title={t("goals.addFallback")}
            onClick={handleAddFallback}
          >
            <IconGoalFallback size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  );
};

export default StepEditorFlowConfiguration;
