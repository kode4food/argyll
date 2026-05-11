import { HTTPMethod, StepType } from "@/app/api";
import { Attribute, ValidationError } from "./stepEditorTypes";
import {
  validateAttributesList,
  validateMappings,
} from "./stepAttrValidationUtils";

export { validateAttributesList } from "./stepAttrValidationUtils";

export function parseFlowGoals(value: string): string[] {
  return value
    .split(/[\n,]+/)
    .map((goal) => goal.trim())
    .filter((goal) => goal.length > 0);
}

const endpointParamPattern = /\{([^{}]+)\}/g;

function validateGetEndpointParams(
  attributes: Attribute[],
  endpoint: string
): ValidationError | null {
  const params = new Set<string>();
  for (const match of endpoint.matchAll(endpointParamPattern)) {
    if (match[1]) {
      params.add(match[1]);
    }
  }

  for (const attr of attributes) {
    if (attr.attrType !== "input") {
      continue;
    }
    const paramName = attr.mappingName?.trim() || attr.name.trim();
    if (!paramName || params.has(paramName)) {
      continue;
    }
    return {
      key: "stepEditor.getEndpointParamRequired",
      vars: { name: paramName },
    };
  }

  return null;
}

function validateFlowStepConfig(flowGoals: string): ValidationError | null {
  if (parseFlowGoals(flowGoals).length === 0) {
    return { key: "stepEditor.flowGoalsRequired" };
  }
  return null;
}

interface HttpStepConfig {
  endpoint: string;
  httpMethod: HTTPMethod;
  httpTimeout: number;
}

function validateHttpStepConfig(
  attributes: Attribute[],
  http: HttpStepConfig
): ValidationError | null {
  if (!http.endpoint.trim()) {
    return { key: "stepEditor.endpointRequired" };
  }
  if (http.httpMethod === "GET") {
    const endpointError = validateGetEndpointParams(attributes, http.endpoint);
    if (endpointError) return endpointError;
  }
  if (!http.httpTimeout || http.httpTimeout <= 0) {
    return { key: "stepEditor.timeoutPositive" };
  }
  return null;
}

export function getValidationError({
  isCreateMode,
  stepId,
  attributes,
  stepType,
  script,
  endpoint,
  httpMethod,
  httpTimeout,
  flowGoals,
}: {
  isCreateMode: boolean;
  stepId: string;
  attributes: Attribute[];
  stepType: StepType;
  script: string;
  endpoint: string;
  httpMethod: HTTPMethod;
  httpTimeout: number;
  flowGoals: string;
}): ValidationError | null {
  if (isCreateMode && !stepId.trim()) {
    return { key: "stepEditor.stepIdRequired" };
  }
  const attrError = validateAttributesList(attributes);
  if (attrError) return attrError;
  const mappingError = validateMappings(attributes);
  if (mappingError) return mappingError;

  if (stepType === "flow") {
    return validateFlowStepConfig(flowGoals);
  }
  if (stepType === "script") {
    return script.trim() ? null : { key: "stepEditor.scriptRequired" };
  }
  return validateHttpStepConfig(attributes, {
    endpoint,
    httpMethod,
    httpTimeout,
  });
}
