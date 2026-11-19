import {
  ActionMode,
  AgentConfig,
  AttributeSpec,
  DEFAULT_HTTP_METHOD,
  fromGoalSets,
  GoalSets,
  Handling,
  HTTPMethod,
  Step,
  StepType,
} from "@/app/api";
import { nonEmptyGoalSets } from "@/utils/goalSets";

export type {
  AttributeRoleType,
  Attribute,
  ValidationError,
} from "./stepEditorTypes";
export {
  buildAttributesFromStep,
  createStepAttributes,
} from "./stepAttributeUtils";
export {
  getValidationError,
  validateAttributesList,
} from "./stepValidationUtils";

export function normalizeHttpMethod(method?: string): HTTPMethod {
  return method === "GET" || method === "PUT" || method === "DELETE"
    ? method
    : DEFAULT_HTTP_METHOD;
}

export function buildStepPayload({
  stepId,
  name,
  description,
  stepType,
  attributes,
  tags,
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
}: {
  stepId: string;
  name: string;
  description: string;
  stepType: StepType;
  attributes: Record<string, AttributeSpec>;
  tags: string[] | undefined;
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
  flowGoals: GoalSets;
  flowSpaceId: string;
  agent: AgentConfig;
  handling: Handling;
}): Step {
  const stepData: Step = {
    id: stepId.trim(),
    name,
    description: description.trim() || undefined,
    type: stepType,
    attributes,
    tags,
    predicate: predicate.trim()
      ? {
          language: predicateLanguage,
          source: predicate.trim(),
        }
      : undefined,
    handling: handling === "standard" ? undefined : handling,
  };

  if (stepType === "agent") {
    stepData.agent = agentPayload(agent);
    return stepData;
  }
  if (stepType === "flow") {
    stepData.flow = {
      goals: fromGoalSets(nonEmptyGoalSets(flowGoals)),
      space_id: flowSpaceId || undefined,
    };
    stepData.http = undefined;
    stepData.script = undefined;
  } else if (stepType === "script") {
    stepData.script = {
      language: scriptLanguage,
      source: script.trim(),
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

function agentPayload(agent: AgentConfig): AgentConfig {
  const mcp = (agent.mcp ?? [])
    .map((s) => ({ server: s.server.trim(), tools: s.tools }))
    .filter((s) => s.server !== "");
  return {
    provider: agent.provider.trim(),
    model: agent.model.trim(),
    effort: agent.effort || undefined,
    prologue: agent.prologue,
    max_turns: agent.max_turns,
    max_tokens: agent.max_tokens,
    timeout: agent.timeout || undefined,
    mcp: mcp.length > 0 ? mcp : undefined,
  };
}
