import { useCallback, useState } from "react";
import { ArgyllApi, toGoalSets, Step } from "@/app/api";
import { formatJSON } from "@/utils/jsonUtils";
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

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

function isAgentDraft(value: unknown): boolean {
  if (value == null) return true;
  if (!isObject(value)) return false;
  const validText = ["provider", "model", "prologue"].every(
    (key) => value[key] === undefined || typeof value[key] === "string"
  );
  const validNumbers = ["max_turns", "max_tokens", "timeout"].every(
    (key) => value[key] === undefined || typeof value[key] === "number"
  );
  if (!validText || !validNumbers) return false;
  const { mcp } = value;
  if (mcp == null) return true;
  return Array.isArray(mcp) && mcp.every(isMCPSelection);
}

function isMCPSelection(value: unknown): boolean {
  if (!isObject(value) || typeof value.server !== "string") return false;
  const { tools } = value;
  const validTools =
    Array.isArray(tools) && tools.every((tool) => typeof tool === "string");
  return tools == null || validTools;
}

export const parseJsonStep = (
  rawValue: string,
  t: TFn
): { step: Step } | { error: string } => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(rawValue);
  } catch {
    return { error: t("stepEditor.invalidJson") };
  }
  if (!isObject(parsed)) {
    return { error: t("stepEditor.invalidJsonObject") };
  }
  if (!isAgentDraft(parsed.agent)) {
    return { error: t("stepEditor.invalidJsonAgent") };
  }
  return { step: parsed as unknown as Step };
};

interface UseStepPersistenceArgs {
  isCreateMode: boolean;
  stepId: string;
  buildStepData: () => Step;
  onUpdate: (step: Step) => void;
  onClose: () => void;
  t: TFn;
}

export function useStepPersistence({
  isCreateMode,
  stepId,
  buildStepData,
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
        script: stepData.script?.source || "",
        endpoint: stepData.http?.invoke?.endpoint || "",
        httpMethod: normalizeHttpMethod(stepData.http?.invoke?.method),
        httpTimeout: stepData.http?.invoke?.timeout || 0,
        flowGoals: toGoalSets(stepData.flow?.goals),
        agent: stepData.agent,
        handling: stepData.handling || "standard",
        compensateEndpoint: stepData.http?.compensate?.endpoint || "",
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
        setError(err.message || t("stepEditor.saveFailed"));
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

  const getSerializedStepData = useCallback(
    () => formatJSON(buildStepData()),
    [buildStepData]
  );

  return {
    saving,
    error,
    setError,
    handleSave,
    handleJsonSave,
    getSerializedStepData,
  };
}
