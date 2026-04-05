import React from "react";
import { useUI } from "@/app/contexts/UIContext";
import { AttributeType } from "@/app/api";
import { useFlowCreation } from "@/app/contexts/FlowCreationContext";
import { useFlowFormScrollFade } from "./useFlowFormScrollFade";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";
import {
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  FlowInputStatus,
  getFlowInputStatus,
  validateJsonString,
} from "./flowFormUtils";
import { useT } from "@/app/i18n";
import {
  FlowInputOption,
  getFlowPlanAttributeOptions,
} from "@/utils/flowPlanAttributeOptions";
import FlowGoalsSection from "./FlowGoalsSection";
import FlowAttributesSection from "./FlowAttributesSection";
import FlowStartSection from "./FlowStartSection";
import styles from "./FlowCreateForm.module.css";

interface FlowCreateFormProps {
  onCreateStep?: () => void;
}

const getTypePlaceholder = (type?: AttributeType): string => {
  switch (type) {
    case AttributeType.Number:
      return "0";
    case AttributeType.Boolean:
      return "false";
    case AttributeType.Object:
      return "{}";
    case AttributeType.Array:
      return "[]";
    case AttributeType.String:
      return '""';
    case AttributeType.Null:
      return "null";
    default:
      return "";
  }
};

const getFlowInputPlaceholder = (option: FlowInputOption): string => {
  if (option.defaultValue !== undefined) {
    return option.defaultValue;
  }

  return getTypePlaceholder(option.type);
};

const FlowCreateForm: React.FC<FlowCreateFormProps> = ({ onCreateStep }) => {
  const {
    newID,
    setNewID,
    setIDManuallyEdited,
    handleStepChange,
    initialState,
    setInitialState,
    creating,
    handleCreateFlow,
    steps,
    generateID,
    sortSteps,
  } = useFlowCreation();
  const {
    previewPlan,
    goalSteps,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  } = useUI();
  const t = useT();

  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  React.useEffect(() => {
    if (editorMode !== "basic") {
      setFocusedPreviewAttribute(null);
    }
  }, [editorMode, setFocusedPreviewAttribute]);

  const sortedSteps = React.useMemo(() => sortSteps(steps), [steps, sortSteps]);

  const { sidebarListRef, showTopFade, showBottomFade } =
    useFlowFormScrollFade(true);

  const { included, satisfied, missingByStep } = useFlowFormStepFiltering(
    steps,
    initialState,
    previewPlan
  );
  const { flowInputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(previewPlan),
    [previewPlan]
  );
  const emptyAttributesLabel =
    goalSteps.length === 0
      ? t("flowCreate.noGoalsSelected")
      : t("flowCreate.noPotentialInputs");
  const flowInputNames = React.useMemo(
    () => flowInputOptions.map((option) => option.name),
    [flowInputOptions]
  );

  React.useEffect(() => {
    if (editorMode !== "basic") {
      return;
    }

    if (
      focusedPreviewAttribute &&
      !flowInputNames.includes(focusedPreviewAttribute)
    ) {
      setFocusedPreviewAttribute(null);
    }
  }, [
    editorMode,
    flowInputNames,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
  ]);
  const flowInputValuesRaw = React.useMemo(
    () => buildInitialStateInputValues(initialState, flowInputNames),
    [flowInputNames, initialState]
  );
  const flowInputValues = React.useMemo(() => {
    const values: Record<string, string> = {};

    flowInputOptions.forEach((option) => {
      const rawValue = flowInputValuesRaw[option.name] || "";
      const status = getFlowInputStatus(option, rawValue);
      values[option.name] = status === "defaulted" ? "" : rawValue;
    });

    return values;
  }, [flowInputOptions, flowInputValuesRaw]);

  const statusLabelByType: Record<FlowInputStatus, string> = {
    provided: t("flowCreate.providedBadge"),
    defaulted: t("flowCreate.defaultBadge"),
    required: t("flowCreate.requiredBadge"),
    optional: t("flowStats.optionalLabel"),
  };
  const handleBasicInputChange = React.useCallback(
    (name: string, value: string) => {
      const option = flowInputOptions.find((item) => item.name === name);
      const normalizedValue =
        value.trim() === "" &&
        option?.required &&
        option.defaultValue !== undefined
          ? option.defaultValue
          : value;
      const nextValues = {
        ...flowInputValuesRaw,
        [name]: normalizedValue,
      };
      const nextState = buildInitialStateFromInputValues(
        nextValues,
        flowInputNames
      );

      setInitialState(JSON.stringify(nextState, null, 2));
    },
    [flowInputNames, flowInputOptions, flowInputValuesRaw, setInitialState]
  );

  return (
    <div className={styles.panel}>
      <div className={styles.container}>
        <div className={styles.main}>
          <div className={styles.panelBody}>
            <FlowGoalsSection
              goalSteps={goalSteps}
              included={included}
              missingByStep={missingByStep}
              onCreateStep={onCreateStep}
              onGoalStepsChange={handleStepChange}
              satisfied={satisfied}
              showBottomFade={showBottomFade}
              showTopFade={showTopFade}
              sidebarListRef={sidebarListRef}
              sortedSteps={sortedSteps}
              stepsCount={steps.length}
            />

            <FlowAttributesSection
              editorMode={editorMode}
              emptyAttributesLabel={emptyAttributesLabel}
              flowInputOptions={flowInputOptions}
              flowInputValues={flowInputValues}
              flowInputValuesRaw={flowInputValuesRaw}
              getFlowInputPlaceholder={getFlowInputPlaceholder}
              handleBasicInputChange={handleBasicInputChange}
              initialState={initialState}
              jsonError={jsonError}
              onEditorModeChange={setEditorMode}
              onFocusedPreviewAttributeChange={setFocusedPreviewAttribute}
              setInitialState={setInitialState}
              statusLabelByType={statusLabelByType}
              toFlowInputStatus={getFlowInputStatus}
            />
          </div>

          <div className={styles.panelFooter}>
            <FlowStartSection
              creating={creating}
              disableStart={
                creating ||
                !newID.trim() ||
                goalSteps.length === 0 ||
                (editorMode === "json" && jsonError !== null)
              }
              flowId={newID}
              onCreateFlow={handleCreateFlow}
              onFlowIdChange={(value) => {
                setNewID(value);
                setIDManuallyEdited(true);
              }}
              onGenerateId={() => {
                setNewID(generateID());
                setIDManuallyEdited(false);
              }}
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
