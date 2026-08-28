import {
  ActionMode,
  AttributeSpec,
  Handling,
  HTTPMethod,
  Step,
  StepType,
} from "@/app/api";
import { parseFlowGoals } from "./stepValidationUtils";

export type {
  AttributeRoleType,
  Attribute,
  ValidationError,
} from "./stepEditorTypes";
export {
  buildAttributesFromStep,
  createStepAttributes,
} from "./stepAttributeUtils";
export { createStepLabels } from "./stepLabelUtils";
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
  labels,
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
}: {
  stepId: string;
  name: string;
  stepType: StepType;
  attributes: Record<string, AttributeSpec>;
  labels: Record<string, string> | undefined;
  predicate: string;
  predicateLanguage: string;
  script: string;
  scriptLanguage: string;
  endpoint: string;
  httpMethod: HTTPMethod;
  httpMode: ActionMode;
  healthCheck: string;
  compensate: string;
  compensateMethod: HTTPMethod;
  compensateTimeout: number;
  compensateMode: ActionMode;
  httpTimeout: number;
  flowGoals: string;
  flowCompensate: boolean;
  handling: Handling;
}): Step {
  const stepData: Step = {
    id: stepId.trim(),
    name,
    type: stepType,
    attributes,
    labels,
    predicate: predicate.trim()
      ? {
          language: predicateLanguage,
          script: predicate.trim(),
        }
      : undefined,
    handling: handling === "standard" ? undefined : handling,
  };

  if (stepType === "flow") {
    stepData.flow = {
      goals: parseFlowGoals(flowGoals),
      compensate: flowCompensate,
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
    const compensateEndpoint = compensate.trim();
    stepData.http = {
      invoke: {
        endpoint: endpoint.trim(),
        method: httpMethod,
        timeout: httpTimeout,
        mode: httpMode,
      },
      compensate:
        handling === "compensated" && compensateEndpoint
          ? {
              endpoint: compensateEndpoint,
              method: compensateMethod,
              // omitted means "inherit the invoke timeout"
              ...(compensateTimeout > 0 && { timeout: compensateTimeout }),
              mode: compensateMode,
            }
          : undefined,
      health: healthCheck.trim() || undefined,
    };
    stepData.script = undefined;
    stepData.flow = undefined;
  }

  return stepData;
}
