export type {
  FlowStatus,
  StepStatus,
  HealthStatus,
  StepType,
  AttributeSpec,
  HTTPConfig,
  ScriptConfig,
  WorkConfig,
  Step,
  ExecutionPlan,
  FlowContext,
  ExecutionResult,
  StepHealth,
  AttributeValue,
  WorkState,
} from "./types";

export {
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_LUA,
  AttributeType,
  AttributeRole,
} from "./types";

export { ArgyllApi } from "./client";
import { ArgyllApi } from "./client";

export const api = new ArgyllApi();
