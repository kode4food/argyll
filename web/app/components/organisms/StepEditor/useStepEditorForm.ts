import { useCallback, useMemo, useState } from "react";
import {
  ActionMode,
  AgentConfig,
  toGoalSets,
  GoalSets,
  Handling,
  HTTPMethod,
  SCRIPT_LANGUAGE_ARGYLL,
  SCRIPT_LANGUAGE_LUA,
  Step,
  StepType,
} from "@/app/api";
import {
  buildStepPayload,
  createStepAttributes,
  normalizeHttpMethod,
} from "./stepEditorUtils";
import { useT } from "@/app/i18n";
import { useAttributeList } from "./useAttributeList";
import { EMPTY_AGENT } from "./stepEditorConstants";
import { useStepPersistence } from "./useStepPersistence";

export interface StepEditorFormOptions {
  step: Step | null;
  onUpdate: (updatedStep: Step) => void;
  onClose: () => void;
  defaultTags?: string[];
  draft?: Step;
}

export function useStepEditorForm({
  step,
  onUpdate,
  onClose,
  defaultTags,
  draft,
}: StepEditorFormOptions) {
  const t = useT();
  const isCreateMode = step === null;
  const seed = step ?? draft ?? null;

  const [stepId, setStepId] = useState(seed?.id || "");
  const [name, setName] = useState(seed?.name || "");
  const [description, setDescription] = useState(seed?.description || "");
  const [stepType, setStepTypeState] = useState<StepType>(
    seed?.type || "service"
  );
  const [predicate, setPredicate] = useState(seed?.predicate?.source || "");
  const [predicateLanguage, setPredicateLanguage] = useState(
    seed?.predicate?.language || SCRIPT_LANGUAGE_ARGYLL
  );
  const [endpoint, setEndpoint] = useState(seed?.http?.invoke?.endpoint || "");
  const [httpMethod, setHttpMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(seed?.http?.invoke?.method)
  );
  const [httpMode, setHttpMode] = useState<ActionMode>(
    seed?.http?.invoke?.mode ?? "sync"
  );
  const [healthCheck, setHealthCheck] = useState(seed?.http?.health || "");
  const [compensate, setCompensate] = useState(
    seed?.http?.compensate?.endpoint || ""
  );
  const [compensateMethod, setCompensateMethod] = useState<HTTPMethod>(
    normalizeHttpMethod(seed?.http?.compensate?.method)
  );
  const [compensateTimeout, setCompensateTimeout] = useState(
    seed?.http?.compensate?.timeout || 0
  );
  const [compensateMode, setCompensateMode] = useState<ActionMode>(
    seed?.http?.compensate?.mode ?? "sync"
  );
  const [httpTimeout, setHttpTimeout] = useState(
    seed?.http?.invoke?.timeout ?? 0
  );
  const [script, setScript] = useState(seed?.script?.source || "");
  const [scriptLanguage, setScriptLanguage] = useState(
    seed?.script?.language || SCRIPT_LANGUAGE_LUA
  );
  const [flowGoals, setFlowGoals] = useState<GoalSets>(
    toGoalSets(seed?.flow?.goals)
  );
  const [flowSpaceId, setFlowSpaceId] = useState(seed?.flow?.space_id || "");
  const [agent, setAgent] = useState<AgentConfig>(seed?.agent ?? EMPTY_AGENT);
  const [handling, setHandling] = useState<Handling>(
    seed?.handling || "standard"
  );

  const {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    resetAttributes,
    clearCompensated,
  } = useAttributeList(seed, t);

  const [tags, setTags] = useState<string[]>(
    () => seed?.tags ?? defaultTags ?? []
  );

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
      description,
      stepType,
      attributes: stepAttributes,
      tags: tags.length > 0 ? tags : undefined,
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
      flowSpaceId,
      agent,
      handling,
    });
  }, [
    agent,
    attributes,
    tags,
    compensate,
    compensateMethod,
    compensateTimeout,
    compensateMode,
    endpoint,
    flowGoals,
    flowSpaceId,
    healthCheck,
    httpMethod,
    httpMode,
    httpTimeout,
    handling,
    name,
    description,
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
      setDescription(stepData.description || "");
      setStepTypeState(stepData.type || "service");
      setPredicate(stepData.predicate?.source || "");
      setPredicateLanguage(
        stepData.predicate?.language || SCRIPT_LANGUAGE_ARGYLL
      );
      setScript(stepData.script?.source || "");
      setScriptLanguage(stepData.script?.language || SCRIPT_LANGUAGE_LUA);
      setFlowGoals(toGoalSets(stepData.flow?.goals));
      setFlowSpaceId(stepData.flow?.space_id || "");
      setAgent({ ...EMPTY_AGENT, ...stepData.agent });
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
      setHttpTimeout(stepData.http?.invoke?.timeout ?? 0);
      setHandling(stepData.handling || "standard");
      resetAttributes(stepData);
      setTags(stepData?.tags ?? []);
    },
    [resetAttributes]
  );

  const {
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    getSerializedStepData,
  } = useStepPersistence({
    isCreateMode,
    stepId,
    buildStepData,
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
    description,
    setDescription,
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
    flowSpaceId,
    setFlowSpaceId,
    agent,
    setAgent,
    handling,
    setHandling: changeHandling,
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    tags,
    setTags,
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    getSerializedStepData,
    applyStepData,
    isCreateMode,
    contextValue,
  };
}
