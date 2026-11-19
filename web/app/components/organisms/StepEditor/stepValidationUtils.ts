import {
  AgentConfig,
  GoalSets,
  Handling,
  HTTPMethod,
  StepType,
} from "@/app/api";
import { nonEmptyGoalSets } from "@/utils/goalSets";
import { Attribute, ValidationError } from "./stepEditorTypes";
import {
  validateAttributesList,
  validateMappings,
} from "./stepAttrValidationUtils";

export { validateAttributesList } from "./stepAttrValidationUtils";

const ENDPOINT_PARAM_PATTERN = /\{([^{}]+)\}/g;

function validateGetEndpointParams(
  attributes: Attribute[],
  endpoint: string
): ValidationError | null {
  const params = new Set<string>();
  for (const match of endpoint.matchAll(ENDPOINT_PARAM_PATTERN)) {
    if (match[1]) {
      params.add(match[1]);
    }
  }

  for (const attr of attributes) {
    if (attr.role !== "required") {
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

function validateFlowStepConfig(flowGoals: GoalSets): ValidationError | null {
  if (nonEmptyGoalSets(flowGoals).length === 0) {
    return { key: "stepEditor.flowGoalsRequired" };
  }
  return null;
}

function validateCompensation(
  attributes: Attribute[],
  handling: Handling,
  endpoint: string
): ValidationError | null {
  if (handling === "compensated" && !endpoint.trim()) {
    return { key: "stepEditor.compensateEndpointRequired" };
  }

  const compensated = attributes.filter((attr) => attr.compensated);
  if (handling !== "compensated" && compensated.length > 0) {
    return { key: "stepEditor.compensatedAttributeHandlingRequired" };
  }

  const names = new Set<string>();
  for (const attr of compensated) {
    const name = attr.mappingName?.trim() || attr.name.trim();
    if (names.has(name)) {
      return {
        key: "stepEditor.compensatedAttributeConflict",
        vars: { name },
      };
    }
    names.add(name);
  }
  return null;
}

const AGENT_REQUIRED_TEXT: [keyof AgentConfig, string][] = [
  ["provider", "stepEditor.agentProviderRequired"],
  ["model", "stepEditor.agentModelRequired"],
  ["prologue", "stepEditor.agentPrologueRequired"],
];

function validateAgentStepConfig(
  agent: AgentConfig | undefined
): ValidationError | null {
  for (const [field, key] of AGENT_REQUIRED_TEXT) {
    if (!String(agent?.[field] ?? "").trim()) return { key };
  }
  if (!agent || !(agent.max_turns > 0)) {
    return { key: "stepEditor.agentMaxTurnsPositive" };
  }
  if (!(agent.max_tokens > 0)) {
    return { key: "stepEditor.agentMaxTokensPositive" };
  }
  const names = new Set<string>();
  for (const { server } of agent.mcp ?? []) {
    const name = server.trim();
    if (!name) return { key: "stepEditor.agentMCPServerRequired" };
    if (names.has(name)) {
      return { key: "stepEditor.agentMCPServerDuplicate", vars: { name } };
    }
    names.add(name);
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
  if (http.httpTimeout < 0) {
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
  agent,
  handling,
  compensateEndpoint,
}: {
  isCreateMode: boolean;
  stepId: string;
  attributes: Attribute[];
  stepType: StepType;
  script: string;
  endpoint: string;
  httpMethod: HTTPMethod;
  httpTimeout: number;
  flowGoals: GoalSets;
  agent?: AgentConfig;
  handling: Handling;
  compensateEndpoint: string;
}): ValidationError | null {
  if (isCreateMode && !stepId.trim()) {
    return { key: "stepEditor.stepIdRequired" };
  }
  const attrError = validateAttributesList(attributes);
  if (attrError) return attrError;
  const mappingError = validateMappings(attributes);
  if (mappingError) return mappingError;
  const compensationError = validateCompensation(
    attributes,
    handling,
    compensateEndpoint
  );
  if (compensationError) return compensationError;

  if (stepType === "flow") {
    return validateFlowStepConfig(flowGoals);
  }
  if (stepType === "agent") {
    return validateAgentStepConfig(agent);
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
