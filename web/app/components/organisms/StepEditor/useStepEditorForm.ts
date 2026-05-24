import { useCallback, useMemo, useState } from "react";
import { HTTPMethod, SCRIPT_LANGUAGE_LUA, Step, StepType } from "@/app/api";
import {
  buildStepPayload,
  createStepAttributes,
  normalizeHttpMethod,
} from "./stepEditorUtils";
import { useT } from "@/app/i18n";
import { useAttributeList } from "./useAttributeList";
import { useStepPersistence } from "./useStepPersistence";

export function useStepEditorForm(
  step: Step | null,
  onUpdate: (updatedStep: Step) => void,
  onClose: () => void
) {
  const t = useT();
  const isCreateMode = step === null;

  const [stepId, setStepId] = useState(step?.id || "");
  const [name, setName] = useState(step?.name || "");
  const [stepType, setStepType] = useState<StepType>(step?.type || "sync");
  const [predicate, setPredicate] = useState(step?.predicate?.script || "");
  const [predicateLanguage, setPredicateLanguage] = useState(
    step?.predicate?.language || SCRIPT_LANGUAGE_LUA
  );
  const [endpoint, setEndpoint] = useState(step?.http?.endpoint || "");
  const [httpMethod, setHttpMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(step?.http?.method)
  );
  const [healthCheck, setHealthCheck] = useState(
    step?.http?.health_check || ""
  );
  const [compensate, setCompensate] = useState(step?.http?.compensate || "");
  const [httpTimeout, setHttpTimeout] = useState(
    step &&
      (step.type === "sync" || step.type === "async") &&
      step.http?.timeout
      ? step.http.timeout
      : 5000
  );
  const [script, setScript] = useState(step?.script?.script || "");
  const [scriptLanguage, setScriptLanguage] = useState(
    step?.script?.language || SCRIPT_LANGUAGE_LUA
  );
  const [flowGoals, setFlowGoals] = useState(
    step?.flow?.goals?.join(", ") || ""
  );
  const [memoizable, setMemoizable] = useState(step?.memoizable || false);

  const {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    resetAttributes,
  } = useAttributeList(step, t);

  const buildStepData = useCallback((): Step => {
    const stepAttributes = createStepAttributes(attributes);
    return buildStepPayload({
      stepId,
      name,
      stepType,
      attributes: stepAttributes,
      predicate,
      predicateLanguage,
      script,
      scriptLanguage,
      endpoint,
      httpMethod,
      healthCheck,
      compensate,
      httpTimeout,
      flowGoals,
      memoizable,
    });
  }, [
    attributes,
    compensate,
    endpoint,
    flowGoals,
    healthCheck,
    httpMethod,
    httpTimeout,
    memoizable,
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
      setStepType(stepData.type || "sync");
      setPredicate(stepData.predicate?.script || "");
      setPredicateLanguage(stepData.predicate?.language || SCRIPT_LANGUAGE_LUA);
      setScript(stepData.script?.script || "");
      setScriptLanguage(stepData.script?.language || SCRIPT_LANGUAGE_LUA);
      setFlowGoals(stepData.flow?.goals?.join(", ") || "");
      setEndpoint(stepData.http?.endpoint || "");
      setHttpMethod(normalizeHttpMethod(stepData.http?.method));
      setHealthCheck(stepData.http?.health_check || "");
      setCompensate(stepData.http?.compensate || "");
      setHttpTimeout(stepData.http?.timeout || 5000);
      setMemoizable(Boolean(stepData.memoizable));
      resetAttributes(stepData);
    },
    [resetAttributes]
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
      setStepType,
      attributes,
      addAttribute,
      updateAttribute,
      removeAttribute,
      endpoint,
      setEndpoint,
      httpMethod,
      setHttpMethod,
      healthCheck,
      setHealthCheck,
      compensate,
      setCompensate,
      httpTimeout,
      setHttpTimeout,
      flowGoals,
      setFlowGoals,
      memoizable,
      setMemoizable,
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
      healthCheck,
      compensate,
      httpTimeout,
      flowGoals,
      memoizable,
    ]
  );

  return {
    stepId,
    setStepId,
    name,
    setName,
    stepType,
    setStepType,
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    endpoint,
    setEndpoint,
    httpMethod,
    setHttpMethod,
    healthCheck,
    setHealthCheck,
    compensate,
    setCompensate,
    httpTimeout,
    setHttpTimeout,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    flowGoals,
    setFlowGoals,
    memoizable,
    setMemoizable,
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
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
