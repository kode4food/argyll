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
      if (vars?.reason && true) {
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

  const handleSave = async () => {
    const validationError = getValidationError({
      isCreateMode,
      stepId,
      attributes,
      stepType,
      script,
      endpoint,
      httpTimeout,
      flowGoals,
    });

    if (validationError) {
      setError(formatValidationError(validationError));
      return;
    }

    setSaving(true);
    setError(null);

    try {
      const api = new ArgyllApi();

      const stepAttributes = createStepAttributes(attributes);
      const { inputMap, outputMap } = buildFlowMaps(attributes);
      const stepData = buildStepPayload({
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
  };

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
    isCreateMode,
    contextValue,
  };
}
