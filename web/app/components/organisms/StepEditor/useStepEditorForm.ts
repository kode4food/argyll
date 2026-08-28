import { useCallback, useMemo, useState } from "react";
import {
  ActionMode,
  Handling,
  HTTPMethod,
  SCRIPT_LANGUAGE_LUA,
  Step,
  StepType,
} from "@/app/api";
import {
  buildStepPayload,
  createStepAttributes,
  createStepLabels,
  normalizeHttpMethod,
} from "./stepEditorUtils";
import { useT } from "@/app/i18n";
import { useAttributeList } from "./useAttributeList";
import { useLabelList } from "./useLabelList";
import { useStepPersistence } from "./useStepPersistence";

export interface StepEditorFormOptions {
  step: Step | null;
  onUpdate: (updatedStep: Step) => void;
  onClose: () => void;
  defaultLabels?: Record<string, string>;
}

export function useStepEditorForm({
  step,
  onUpdate,
  onClose,
  defaultLabels,
}: StepEditorFormOptions) {
  const t = useT();
  const isCreateMode = step === null;

  const [stepId, setStepId] = useState(step?.id || "");
  const [name, setName] = useState(step?.name || "");
  const [stepType, setStepTypeState] = useState<StepType>(
    step?.type || "service"
  );
  const [predicate, setPredicate] = useState(step?.predicate?.script || "");
  const [predicateLanguage, setPredicateLanguage] = useState(
    step?.predicate?.language || SCRIPT_LANGUAGE_LUA
  );
  const [endpoint, setEndpoint] = useState(step?.http?.invoke?.endpoint || "");
  const [httpMethod, setHttpMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(step?.http?.invoke?.method)
  );
  const [httpMode, setHttpMode] = useState<ActionMode>(
    step?.http?.invoke?.mode ?? "sync"
  );
  const [healthCheck, setHealthCheck] = useState(step?.http?.health || "");
  const [compensate, setCompensate] = useState(
    step?.http?.compensate?.endpoint || ""
  );
  const [compensateMethod, setCompensateMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(step?.http?.compensate?.method)
  );
  const [compensateTimeout, setCompensateTimeout] = useState(
    step?.http?.compensate?.timeout || 0
  );
  const [compensateMode, setCompensateMode] = useState<ActionMode>(
    step?.http?.compensate?.mode ?? "sync"
  );
  const [httpTimeout, setHttpTimeout] = useState(
    step && step.type === "service" && step.http?.invoke?.timeout
      ? step.http.invoke.timeout
      : 5000
  );
  const [script, setScript] = useState(step?.script?.script || "");
  const [scriptLanguage, setScriptLanguage] = useState(
    step?.script?.language || SCRIPT_LANGUAGE_LUA
  );
  const [flowGoals, setFlowGoals] = useState(
    step?.flow?.goals?.join(", ") || ""
  );
  const [flowCompensate, setFlowCompensate] = useState(
    step?.flow?.compensate || false
  );
  const [handling, setHandling] = useState<Handling>(
    step?.handling || "standard"
  );

  const {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    resetAttributes,
    clearCompensated,
  } = useAttributeList(step, t);

  const { labels, addLabel, updateLabel, removeLabel, resetLabels } =
    useLabelList(step, defaultLabels);

  const changeHandling = useCallback(
    (next: Handling) => {
      setHandling(next);
      if (next !== "compensated") clearCompensated();
    },
    [clearCompensated]
  );

  const changeStepType = useCallback(
    (next: StepType) => {
      setStepTypeState(next);
      if (next !== "service") {
        changeHandling("standard");
      }
    },
    [changeHandling]
  );

  const buildStepData = useCallback((): Step => {
    const stepAttributes = createStepAttributes(attributes);
    return buildStepPayload({
      stepId,
      name,
      stepType,
      attributes: stepAttributes,
      labels: createStepLabels(labels),
      predicate,
      predicateLanguage,
      script,
      scriptLanguage,
      endpoint,
      httpMethod,
      httpMode,
      healthCheck,
      compensate,
      compensateMethod,
      compensateTimeout,
      compensateMode,
      httpTimeout,
      flowGoals,
      flowCompensate,
      handling,
    });
  }, [
    attributes,
    labels,
    compensate,
    compensateMethod,
    compensateTimeout,
    compensateMode,
    endpoint,
    flowCompensate,
    flowGoals,
    healthCheck,
    httpMethod,
    httpMode,
    httpTimeout,
    handling,
    name,
    predicate,
    predicateLanguage,
    script,
    scriptLanguage,
    stepId,
    stepType,
  ]);

  const applyStepData = useCallback(
    (stepData: Step) => {
      setStepId(stepData.id || "");
      setName(stepData.name || "");
      setStepTypeState(stepData.type || "service");
      setPredicate(stepData.predicate?.script || "");
      setPredicateLanguage(stepData.predicate?.language || SCRIPT_LANGUAGE_LUA);
      setScript(stepData.script?.script || "");
      setScriptLanguage(stepData.script?.language || SCRIPT_LANGUAGE_LUA);
      setFlowGoals(stepData.flow?.goals?.join(", ") || "");
      setFlowCompensate(Boolean(stepData.flow?.compensate));
      setEndpoint(stepData.http?.invoke?.endpoint || "");
      setHttpMethod(normalizeHttpMethod(stepData.http?.invoke?.method));
      setHttpMode(stepData.http?.invoke?.mode ?? "sync");
      setHealthCheck(stepData.http?.health || "");
      setCompensate(stepData.http?.compensate?.endpoint || "");
      setCompensateMethod(
        normalizeHttpMethod(stepData.http?.compensate?.method)
      );
      setCompensateTimeout(stepData.http?.compensate?.timeout || 0);
      setCompensateMode(stepData.http?.compensate?.mode ?? "sync");
      setHttpTimeout(stepData.http?.invoke?.timeout || 5000);
      setHandling(stepData.handling || "standard");
      resetAttributes(stepData);
      resetLabels(stepData);
    },
    [resetAttributes, resetLabels]
  );

  const {
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    validateJsonDraft,
    getSerializedStepData,
  } = useStepPersistence({
    isCreateMode,
    stepId,
    buildStepData,
    applyStepData,
    onUpdate,
    onClose,
    t,
  });

  const contextValue = useMemo(
    () => ({
      stepId,
      name,
      stepType,
      isCreateMode,
      setStepId,
      setName,
      setStepType: changeStepType,
      attributes,
      addAttribute,
      updateAttribute,
      removeAttribute,
      endpoint,
      setEndpoint,
      httpMethod,
      setHttpMethod,
      httpMode,
      setHttpMode,
      healthCheck,
      setHealthCheck,
      compensate,
      setCompensate,
      compensateMethod,
      setCompensateMethod,
      compensateTimeout,
      setCompensateTimeout,
      compensateMode,
      setCompensateMode,
      httpTimeout,
      setHttpTimeout,
      flowGoals,
      setFlowGoals,
      flowCompensate,
      setFlowCompensate,
      handling,
      setHandling: changeHandling,
    }),
    [
      stepId,
      name,
      stepType,
      isCreateMode,
      attributes,
      addAttribute,
      updateAttribute,
      removeAttribute,
      endpoint,
      httpMethod,
      httpMode,
      healthCheck,
      compensate,
      compensateMethod,
      compensateTimeout,
      compensateMode,
      httpTimeout,
      flowGoals,
      flowCompensate,
      handling,
      changeHandling,
      changeStepType,
    ]
  );

  return {
    stepId,
    setStepId,
    name,
    setName,
    stepType,
    setStepType: changeStepType,
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    endpoint,
    setEndpoint,
    httpMethod,
    setHttpMethod,
    httpMode,
    setHttpMode,
    healthCheck,
    setHealthCheck,
    compensate,
    setCompensate,
    compensateMethod,
    setCompensateMethod,
    compensateTimeout,
    setCompensateTimeout,
    compensateMode,
    setCompensateMode,
    httpTimeout,
    setHttpTimeout,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    flowGoals,
    setFlowGoals,
    flowCompensate,
    setFlowCompensate,
    handling,
    setHandling: changeHandling,
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    labels,
    addLabel,
    updateLabel,
    removeLabel,
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    validateJsonDraft,
    getSerializedStepData,
    applyStepData,
    isCreateMode,
    contextValue,
  };
}
