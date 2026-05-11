import { AttributeSpec, HTTPMethod, Step, StepType } from "@/app/api";
import { parseFlowGoals } from "./stepValidationUtils";

export type {
  AttributeRoleType,
  Attribute,
  ValidationError,
} from "./stepEditorTypes";
export {
  buildAttributesFromStep,
  createStepAttributes,
  getAttributeIconProps,
} from "./stepAttributeUtils";
export {
  getValidationError,
  parseFlowGoals,
  validateAttributesList,
} from "./stepValidationUtils";

const HTTP_METHOD_POST: HTTPMethod = "POST";

export function normalizeHttpMethod(method?: string): HTTPMethod {
  return method === "GET" || method === "PUT" || method === "DELETE"
    ? method
    : HTTP_METHOD_POST;
}

export function buildStepPayload({
  stepId,
  name,
  stepType,
  attributes,
  predicate,
  predicateLanguage,
  script,
  scriptLanguage,
  endpoint,
  httpMethod,
  healthCheck,
  httpTimeout,
  flowGoals,
  memoizable,
}: {
  stepId: string;
  name: string;
  stepType: StepType;
  attributes: Record<string, AttributeSpec>;
  predicate: string;
  predicateLanguage: string;
  script: string;
  scriptLanguage: string;
  endpoint: string;
  httpMethod: HTTPMethod;
  healthCheck: string;
  httpTimeout: number;
  flowGoals: string;
  memoizable: boolean;
}): Step {
  const stepData: Step = {
    id: stepId.trim(),
    name,
    type: stepType,
    attributes,
    predicate: predicate.trim()
      ? {
          language: predicateLanguage,
          script: predicate.trim(),
        }
      : undefined,
    memoizable,
  };

  if (stepType === "flow") {
    stepData.flow = {
      goals: parseFlowGoals(flowGoals),
    };
    stepData.http = undefined;
    stepData.script = undefined;
  } else if (stepType === "script") {
    stepData.script = {
      language: scriptLanguage,
      script: script.trim(),
    };
    stepData.http = undefined;
    stepData.flow = undefined;
  } else {
    stepData.http = {
      endpoint: endpoint.trim(),
      method: httpMethod,
      health_check: healthCheck.trim() || undefined,
      timeout: httpTimeout,
    };
    stepData.script = undefined;
    stepData.flow = undefined;
  }

  return stepData;
}
