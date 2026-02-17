export type {
  FlowStatus,
  StepStatus,
  HealthStatus,
  StepType,
  AttributeSpec,
  HTTPConfig,
  ScriptConfig,
  FlowConfig,
  WorkConfig,
  Step,
  ExecutionPlan,
  FlowContext,
  QueryFlowsItem,
  QueryFlowsResponse,
  ExecutionResult,
  StepHealth,
  AttributeValue,
  WorkState,
  FlowSort,
  QueryFlowsRequest,
} from "./types";

export {
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
  AttributeType,
  AttributeRole,
} from "./types";

export { ArgyllApi } from "./client";
import { ArgyllApi } from "./client";

export const api = new ArgyllApi();
