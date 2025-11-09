export type {
  WorkflowStatus,
  StepStatus,
  HealthStatus,
  StepType,
  AttributeSpec,
  HTTPConfig,
  ScriptConfig,
  WorkConfig,
  Step,
  ExecutionPlan,
  StepInfo,
  WorkflowContext,
  ExecutionResult,
  StepHealth,
} from "./types";

export {
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_LUA,
  AttributeType,
  AttributeRole,
} from "./types";

export { SpudsApi } from "./client";
import { SpudsApi } from "./client";

export const api = new SpudsApi();
