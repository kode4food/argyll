import { useCallback, useMemo, useRef, useState } from "react";
import {
  ArgyllApi,
  AttributeType,
  SCRIPT_LANGUAGE_LUA,
  Step,
  StepType,
} from "@/app/api";
import {
  Attribute,
  buildAttributesFromStep,
  buildFlowMaps,
  buildStepPayload,
  createStepAttributes,
  getValidationError,
} from "./stepEditorUtils";
import { validateDefaultValue } from "@/utils/stepUtils";
import { useT } from "@/app/i18n";

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
  const [healthCheck, setHealthCheck] = useState(
    step?.http?.health_check || ""
  );
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

  const [attributes, setAttributes] = useState<Attribute[]>(() =>
    buildAttributesFromStep(step)
  );

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const attributeCounterRef = useRef(0);

  const formatValidationError = useCallback(
    (validationError: ReturnType<typeof getValidationError> | null) => {
      if (!validationError) {
        return null;
      }
      const vars = validationError.vars
        ? { ...validationError.vars }
        : undefined;
      if (vars?.reason) {
        vars.reason = t(vars.reason);
      }
      return t(validationError.key, vars);
    },
    [t]
  );

  const addAttribute = useCallback(() => {
    setAttributes((current) => [
      ...current,
      {
        id: `attr-${++attributeCounterRef.current}`,
        attrType: "input",
        name: "",
        dataType: AttributeType.String,
      },
    ]);
  }, []);

  const updateAttribute = useCallback(
    (id: string, field: keyof Attribute, value: any) => {
      setAttributes((current) =>
        current.map((attr) => {
          if (attr.id !== id) return attr;

          const updated = { ...attr, [field]: value };

          if (
            (field === "defaultValue" || field === "dataType") &&
            (updated.attrType === "optional" || updated.attrType === "const") &&
            updated.defaultValue
          ) {
            const validation = validateDefaultValue(
              updated.defaultValue,
              updated.dataType
            );
            updated.validationError = validation.valid
              ? undefined
              : t(validation.errorKey ?? "");
          }

          if (
            field === "attrType" &&
            value !== "optional" &&
            value !== "const"
          ) {
            updated.validationError = undefined;
          }

          if (field === "attrType" && value === "const") {
            updated.forEach = false;
          }

          return updated;
        })
      );
    },
    [t]
  );

  const removeAttribute = useCallback((id: string) => {
    setAttributes((current) => current.filter((attr) => attr.id !== id));
  }, []);

  const cycleAttributeType = useCallback(
    (id: string, currentType: "input" | "optional" | "const" | "output") => {
      const types: ("input" | "optional" | "const" | "output")[] = [
        "input",
        "optional",
        "const",
        "output",
      ];
      const currentIndex = types.indexOf(currentType);
      const nextIndex = (currentIndex + 1) % types.length;
      updateAttribute(id, "attrType", types[nextIndex]);
    },
    [updateAttribute]
  );

  const buildStepData = useCallback((): Step => {
    const stepAttributes = createStepAttributes(attributes);
    const { inputMap, outputMap } = buildFlowMaps(attributes);
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
      healthCheck,
      httpTimeout,
      flowGoals,
      flowInputMap: inputMap,
      flowOutputMap: outputMap,
      memoizable,
    });
  }, [
    attributes,
    endpoint,
    flowGoals,
    healthCheck,
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

  const getStepValidationError = useCallback(
    (stepData: Step) => {
      const mappedAttributes = buildAttributesFromStep(stepData);
      return getValidationError({
        isCreateMode,
        stepId: stepData.id || "",
        attributes: mappedAttributes,
        stepType: stepData.type,
        script: stepData.script?.script || "",
        endpoint: stepData.http?.endpoint || "",
        httpTimeout: stepData.http?.timeout || 0,
        flowGoals: stepData.flow?.goals?.join(", ") || "",
      });
    },
    [isCreateMode]
  );

  const persistStepData = useCallback(
    async (stepData: Step) => {
      const validationError = getStepValidationError(stepData);
      if (validationError) {
        setError(formatValidationError(validationError));
        return;
      }

      setSaving(true);
      setError(null);

      try {
        const api = new ArgyllApi();
        let resultStep: Step;
        if (isCreateMode) {
          resultStep = await api.registerStep(stepData);
        } else {
          resultStep = await api.updateStep(stepId, stepData);
        }
        onUpdate(resultStep);
        onClose();
      } catch (err: any) {
        setError(
          err.response?.data?.error || err.message || t("stepEditor.saveFailed")
        );
      } finally {
        setSaving(false);
      }
    },
    [
      formatValidationError,
      getStepValidationError,
      isCreateMode,
      onClose,
      onUpdate,
      stepId,
      t,
    ]
  );

  const handleSave = async () => {
    await persistStepData(buildStepData());
  };

  const handleJsonSave = async (rawValue: string) => {
    let parsed: unknown;
    try {
      parsed = JSON.parse(rawValue);
    } catch {
      setError(t("stepEditor.invalidJson"));
      return;
    }

    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      setError(t("stepEditor.invalidJsonObject"));
      return;
    }

    await persistStepData(parsed as Step);
  };

  const applyStepDataToForm = useCallback((stepData: Step) => {
    setStepId(stepData.id || "");
    setName(stepData.name || "");
    setStepType(stepData.type || "sync");
    setPredicate(stepData.predicate?.script || "");
    setPredicateLanguage(stepData.predicate?.language || SCRIPT_LANGUAGE_LUA);
    setScript(stepData.script?.script || "");
    setScriptLanguage(stepData.script?.language || SCRIPT_LANGUAGE_LUA);
    setFlowGoals(stepData.flow?.goals?.join(", ") || "");
    setEndpoint(stepData.http?.endpoint || "");
    setHealthCheck(stepData.http?.health_check || "");
    setHttpTimeout(stepData.http?.timeout || 5000);
    setMemoizable(Boolean(stepData.memoizable));
    setAttributes(buildAttributesFromStep(stepData));
    setError(null);
  }, []);

  const validateJsonDraft = useCallback(
    (rawValue: string) => {
      let parsed: unknown;
      try {
        parsed = JSON.parse(rawValue);
      } catch {
        return t("stepEditor.invalidJson");
      }
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        return t("stepEditor.invalidJsonObject");
      }
      const validationError = getStepValidationError(parsed as Step);
      return formatValidationError(validationError);
    },
    [formatValidationError, getStepValidationError, t]
  );

  const getSerializedStepData = useCallback(
    () => JSON.stringify(buildStepData(), null, 2),
    [buildStepData]
  );

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
      cycleAttributeType,
      endpoint,
      setEndpoint,
      healthCheck,
      setHealthCheck,
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
      cycleAttributeType,
      endpoint,
      healthCheck,
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
    healthCheck,
    setHealthCheck,
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
    cycleAttributeType,
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    validateJsonDraft,
    getSerializedStepData,
    applyStepDataToForm,
    isCreateMode,
    contextValue,
  };
}
