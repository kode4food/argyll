import { useCallback, useState } from "react";
import { ArgyllApi, Step } from "@/app/api";
import {
  buildAttributesFromStep,
  getValidationError,
  normalizeHttpMethod,
  ValidationError,
} from "./stepEditorUtils";

type TFn = (key: string, vars?: Record<string, string | number>) => string;

const applyValidationVars = (
  vars: Record<string, string>,
  t: TFn
): Record<string, string> => {
  if (!vars.reason) return vars;
  return { ...vars, reason: t(vars.reason) };
};

const formatValidationError = (
  validationError: ValidationError | null,
  t: TFn
): string | null => {
  if (!validationError) return null;
  const vars = validationError.vars
    ? applyValidationVars({ ...validationError.vars }, t)
    : undefined;
  return t(validationError.key, vars);
};

const parseJsonStep = (
  rawValue: string,
  t: TFn
): { step: Step } | { error: string } => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(rawValue);
  } catch {
    return { error: t("stepEditor.invalidJson") };
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { error: t("stepEditor.invalidJsonObject") };
  }
  return { step: parsed as Step };
};

interface UseStepPersistenceArgs {
  isCreateMode: boolean;
  stepId: string;
  buildStepData: () => Step;
  applyStepDataToForm: (step: Step) => void;
  onUpdate: (step: Step) => void;
  onClose: () => void;
  t: TFn;
}

export function useStepPersistence({
  isCreateMode,
  stepId,
  buildStepData,
  applyStepDataToForm,
  onUpdate,
  onClose,
  t,
}: UseStepPersistenceArgs) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getStepValidationError = useCallback(
    (stepData: Step): ValidationError | null => {
      const mappedAttributes = buildAttributesFromStep(stepData);
      return getValidationError({
        isCreateMode,
        stepId: stepData.id || "",
        attributes: mappedAttributes,
        stepType: stepData.type,
        script: stepData.script?.script || "",
        endpoint: stepData.http?.endpoint || "",
        httpMethod: normalizeHttpMethod(stepData.http?.method),
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
        setError(formatValidationError(validationError, t));
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
    [getStepValidationError, isCreateMode, onClose, onUpdate, stepId, t]
  );

  const handleSave = useCallback(async () => {
    await persistStepData(buildStepData());
  }, [buildStepData, persistStepData]);

  const handleJsonSave = useCallback(
    async (rawValue: string) => {
      const result = parseJsonStep(rawValue, t);
      if ("error" in result) {
        setError(result.error);
        return;
      }
      await persistStepData(result.step);
    },
    [persistStepData, t]
  );

  const validateJsonDraft = useCallback(
    (rawValue: string) => {
      const result = parseJsonStep(rawValue, t);
      if ("error" in result) return result.error;
      const validationError = getStepValidationError(result.step);
      return formatValidationError(validationError, t);
    },
    [getStepValidationError, t]
  );

  const getSerializedStepData = useCallback(
    () => JSON.stringify(buildStepData(), null, 2),
    [buildStepData]
  );

  return {
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    validateJsonDraft,
    getSerializedStepData,
  };
}
