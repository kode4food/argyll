export type {
  FlowStatus,
  StepStatus,
  HealthStatus,
  StepType,
  ActionMode,
  Handling,
  HTTPMethod,
  InputCollect,
  AttributeSpec,
  RequiredConfig,
  OptionalConfig,
  ConstConfig,
  MetaConfig,
  OutputConfig,
  MappingConfig,
  HTTPConfig,
  ScriptConfig,
  FlowConfig,
  WorkConfig,
  Step,
  Space,
  SpacePreviewResponse,
  ExecutionPlan,
  FlowContext,
  FlowSummary,
  QueryFlowsItem,
  QueryFlowsResponse,
  ExecutionResult,
  StepHealth,
  AttributeValue,
  WorkState,
  FlowSort,
  QueryFlowsRequest,
  NodeStepHealth,
} from "./types";

export {
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
  AttributeType,
  AttributeRole,
  META_KEYS,
} from "./types";

export { ArgyllApi } from "./client";
import { ArgyllApi } from "./client";

export const api = new ArgyllApi();
