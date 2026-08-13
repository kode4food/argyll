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
  const [endpoint, setEndpoint] = useState(step?.http?.invoke?.endpoint || "");
  const [httpMethod, setHttpMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(step?.http?.invoke?.method)
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
  const [httpTimeout, setHttpTimeout] = useState(
    step &&
      (step.type === "sync" || step.type === "async") &&
      step.http?.invoke?.timeout
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
      compensateMethod,
      compensateTimeout,
      httpTimeout,
      flowGoals,
      flowCompensate,
      memoizable,
    });
  }, [
    attributes,
    compensate,
    compensateMethod,
    compensateTimeout,
    endpoint,
    flowCompensate,
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
      setFlowCompensate(Boolean(stepData.flow?.compensate));
      setEndpoint(stepData.http?.invoke?.endpoint || "");
      setHttpMethod(normalizeHttpMethod(stepData.http?.invoke?.method));
      setHealthCheck(stepData.http?.health || "");
      setCompensate(stepData.http?.compensate?.endpoint || "");
      setCompensateMethod(
        normalizeHttpMethod(stepData.http?.compensate?.method)
      );
      setCompensateTimeout(stepData.http?.compensate?.timeout || 0);
      setHttpTimeout(stepData.http?.invoke?.timeout || 5000);
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
      compensateMethod,
      setCompensateMethod,
      compensateTimeout,
      setCompensateTimeout,
      httpTimeout,
      setHttpTimeout,
      flowGoals,
      setFlowGoals,
      flowCompensate,
      setFlowCompensate,
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
      compensateMethod,
      compensateTimeout,
      httpTimeout,
      flowGoals,
      flowCompensate,
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
    compensateMethod,
    setCompensateMethod,
    compensateTimeout,
    setCompensateTimeout,
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
