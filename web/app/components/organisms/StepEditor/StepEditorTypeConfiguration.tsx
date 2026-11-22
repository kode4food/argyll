import React from "react";
import { ExecutionPlan, GoalSets, Step } from "@/app/api";
import { useT } from "@/app/i18n";
import { IconStepTypeScript } from "@/utils/iconRegistry";
import ScriptConfigEditor from "./ScriptConfigEditor";
import { useAttributeCompletions } from "./useAttributeCompletions";
import StepEditorAgentConfiguration from "./StepEditorAgentConfiguration";
import StepEditorFlowConfiguration from "./StepEditorFlowConfiguration";
import StepEditorHttpConfiguration from "./StepEditorHttpConfiguration";
import formStyles from "./StepEditorForm.module.css";
import { useStepEditorForm } from "./useStepEditorForm";

export interface FlowPreview {
  previewPlan: ExecutionPlan | null;
  flowInitialState: string;
  setFlowInitialState: (value: string) => void;
  updatePreviewPlan: (
    goalSteps: GoalSets,
    initialState: Record<string, any>,
    spaceId?: string
  ) => Promise<void>;
  clearPreviewPlan: () => void;
}

interface StepEditorTypeConfigurationProps {
  form: ReturnType<typeof useStepEditorForm>;
  steps: Step[];
  flowPreview: FlowPreview;
}

const StepEditorTypeConfiguration: React.FC<
  StepEditorTypeConfigurationProps
> = ({ form, steps, flowPreview }) => {
  const t = useT();
  const { scriptCompletions } = useAttributeCompletions(form.attributes);

  switch (form.stepType) {
    case "script":
      return (
        <ScriptConfigEditor
          Icon={IconStepTypeScript}
          label={t("stepEditor.scriptLabel")}
          completions={scriptCompletions}
          value={form.script}
          onChange={form.setScript}
          language={form.scriptLanguage}
          onLanguageChange={form.setScriptLanguage}
          containerClassName={formStyles.scriptEditorContainer}
        />
      );
    case "agent":
      return (
        <StepEditorAgentConfiguration
          agent={form.agent}
          setAgent={form.setAgent}
        />
      );
    case "flow":
      return (
        <StepEditorFlowConfiguration
          clearPreviewPlan={flowPreview.clearPreviewPlan}
          flowGoals={form.flowGoals}
          flowInitialState={flowPreview.flowInitialState}
          flowSpaceId={form.flowSpaceId}
          previewPlan={flowPreview.previewPlan}
          setFlowGoals={form.setFlowGoals}
          setFlowInitialState={flowPreview.setFlowInitialState}
          setFlowSpaceId={form.setFlowSpaceId}
          stepId={form.stepId}
          steps={steps}
          updatePreviewPlan={flowPreview.updatePreviewPlan}
        />
      );
    default:
      return (
        <StepEditorHttpConfiguration
          endpoint={form.endpoint}
          httpMethod={form.httpMethod}
          httpMode={form.httpMode}
          healthCheck={form.healthCheck}
          compensate={form.compensate}
          compensateMethod={form.compensateMethod}
          compensateTimeout={form.compensateTimeout}
          compensateMode={form.compensateMode}
          setCompensateMode={form.setCompensateMode}
          httpTimeout={form.httpTimeout}
          handling={form.handling}
          stepType={form.stepType}
          setEndpoint={form.setEndpoint}
          setHttpMethod={form.setHttpMethod}
          setHttpMode={form.setHttpMode}
          setHealthCheck={form.setHealthCheck}
          setCompensate={form.setCompensate}
          setCompensateMethod={form.setCompensateMethod}
          setCompensateTimeout={form.setCompensateTimeout}
          setHttpTimeout={form.setHttpTimeout}
        />
      );
  }
};

export default StepEditorTypeConfiguration;
