import React from "react";
import { useUI } from "@/app/contexts/UIContext";
import { AttributeType } from "@/app/api";
import { useFlowCreation } from "@/app/hooks/useFlowCreation";
import { useScrollFade } from "@/app/hooks/useScrollFade";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";
import {
  buildInitialStateFromInputValues,
  buildInitialStateInputValues,
  isAtDefaultValue,
  validateJsonString,
} from "./flowFormUtils";
import { useT } from "@/app/i18n";
import {
  FlowInputOption,
  getFlowPlanAttributeOptions,
} from "@/utils/flowPlanAttributeOptions";
import { generateFlowId } from "@/utils/flowUtils";
import { sortStepsByType } from "@/utils/stepUtils";
import { IconManage, IconSpace } from "@/utils/iconRegistry";
import SpaceManager from "@/app/components/organisms/SpaceManager";
import { useSpaces, useSpaceSelection } from "@/app/store/flowStore";
import FlowGoalsSection from "./FlowGoalsSection";
import SelectField from "@/app/components/molecules/SelectField";
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
    compensate,
    setCompensate,
    creating,
    handleCreateFlow,
    steps,
  } = useFlowCreation();
  const {
    previewPlan,
    goalSteps,
    focusedPreviewAttribute,
    setFocusedPreviewAttribute,
    spaceId,
    setSpaceId,
  } = useUI();
  const spaces = useSpaces();
  const t = useT();

  const [isManagingSpaces, setManagingSpaces] = React.useState(false);
  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const [editorMode, setEditorMode] = React.useState<"basic" | "json">("basic");
  const [flowInputDraftValues, setFlowInputDraftValues] = React.useState<
    Record<string, string>
  >({});

  React.useEffect(() => {
    setJsonError(validateJsonString(initialState));
  }, [initialState]);

  React.useEffect(() => {
    if (editorMode !== "basic") {
      setFocusedPreviewAttribute(null);
      setFlowInputDraftValues({});
    }
  }, [editorMode, setFocusedPreviewAttribute]);

  const spaceSelection = useSpaceSelection();
  const spaceOptions = React.useMemo(
    () => [
      { value: "", label: t("flowCreate.spaceAll"), Icon: IconSpace },
      ...spaces.map((s) => ({
        value: s.id,
        label: s.name,
        title: s.description,
        Icon: IconSpace,
      })),
    ],
    [spaces, t]
  );
  // The engine plans within the Space, so a goal outside it could never run
  const scopedSteps = React.useMemo(() => {
    const selected = spaceId ? spaceSelection[spaceId] : null;
    return selected ? steps.filter((s) => selected.has(s.id)) : steps;
  }, [steps, spaceId, spaceSelection]);
  const sortedSteps = React.useMemo(
    () => sortStepsByType(scopedSteps),
    [scopedSteps]
  );

  const {
    scrollRef: sidebarListRef,
    showTopFade,
    showBottomFade,
  } = useScrollFade();

  const { included, satisfied, blockedByStep, missingByStep } =
    useFlowFormStepFiltering(steps, initialState, previewPlan);
  const { flowInputOptions } = React.useMemo(
    () => getFlowPlanAttributeOptions(previewPlan, steps),
    [previewPlan, steps]
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

    setFlowInputDraftValues((current) => {
      const nextValues: Record<string, string> = {};
      flowInputNames.forEach((name) => {
        if (current[name] !== undefined) {
          nextValues[name] = current[name];
        }
      });
      return nextValues;
    });

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
      const draftValue = flowInputDraftValues[option.name];
      values[option.name] =
        draftValue ?? (isAtDefaultValue(option, rawValue) ? "" : rawValue);
    });

    return values;
  }, [flowInputDraftValues, flowInputOptions, flowInputValuesRaw]);

  const commitBasicInputValues = React.useCallback(() => {
    const nextValues = {
      ...flowInputValuesRaw,
      ...flowInputDraftValues,
    };
    const nextState = buildInitialStateFromInputValues(
      nextValues,
      flowInputNames
    );

    setInitialState(JSON.stringify(nextState, null, 2));
    setFlowInputDraftValues({});
  }, [
    flowInputDraftValues,
    flowInputNames,
    flowInputValuesRaw,
    setInitialState,
  ]);

  const handleBasicInputChange = React.useCallback(
    (name: string, value: string) => {
      setFlowInputDraftValues((current) => ({
        ...current,
        [name]: value,
      }));
    },
    []
  );

  // Excludes the flow ID: disabling its input on an empty ID would make the
  // ID impossible to type
  const isStartSectionDisabled =
    creating ||
    goalSteps.length === 0 ||
    (editorMode === "json" && jsonError !== null);

  const handleEditorModeChange = React.useCallback(
    (mode: "basic" | "json") => {
      if (editorMode === "basic" && mode === "json") {
        commitBasicInputValues();
      }
      setEditorMode(mode);
    },
    [commitBasicInputValues, editorMode]
  );

  return (
    <div className={styles.panel}>
      <div className={styles.container}>
        <div className={styles.main}>
          <div className={styles.panelBody}>
            <div className={styles.spaceRow}>
              <SelectField
                className={styles.spaceSelect}
                ariaLabel={t("flowCreate.spaceLabel")}
                onChange={(value) => setSpaceId(value || null)}
                options={spaceOptions}
                value={spaceId ?? ""}
              />
              <button
                type="button"
                className={styles.spaceManageButton}
                onClick={() => setManagingSpaces(true)}
                title={t("spaceManager.title")}
                aria-label={t("spaceManager.title")}
              >
                <IconManage className={styles.spaceManageIcon} />
              </button>
            </div>

            <SpaceManager
              isOpen={isManagingSpaces}
              onClose={() => setManagingSpaces(false)}
            />

            <FlowGoalsSection
              goalSteps={goalSteps}
              blockedByStep={blockedByStep}
              included={included}
              missingByStep={missingByStep}
              onCreateStep={onCreateStep}
              onGoalStepsChange={handleStepChange}
              satisfied={satisfied}
              showBottomFade={showBottomFade}
              showTopFade={showTopFade}
              sidebarListRef={sidebarListRef}
              sortedSteps={sortedSteps}
              spaceScoped={!!spaceId}
              stepsCount={scopedSteps.length}
            />

            <FlowAttributesSection
              editorMode={editorMode}
              emptyAttributesLabel={emptyAttributesLabel}
              flowInputOptions={flowInputOptions}
              flowInputValues={flowInputValues}
              flowInputValuesRaw={flowInputValuesRaw}
              getFlowInputPlaceholder={getFlowInputPlaceholder}
              handleBasicInputBlur={commitBasicInputValues}
              handleBasicInputChange={handleBasicInputChange}
              initialState={initialState}
              jsonError={jsonError}
              onEditorModeChange={handleEditorModeChange}
              onFocusedPreviewAttributeChange={setFocusedPreviewAttribute}
              setInitialState={setInitialState}
            />
          </div>

          <div className={styles.panelFooter}>
            <FlowStartSection
              compensate={compensate}
              creating={creating}
              disabled={isStartSectionDisabled}
              flowId={newID}
              onCompensateChange={setCompensate}
              onCreateFlow={handleCreateFlow}
              onFlowIdChange={(value) => {
                setNewID(value);
                setIDManuallyEdited(true);
              }}
              onGenerateId={() => {
                setNewID(generateFlowId());
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
