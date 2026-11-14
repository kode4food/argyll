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
  FeedConfig,
  OutputConfig,
  MetricConfig,
  MappingConfig,
  HTTPConfig,
  ScriptConfig,
  FlowConfig,
  Goals,
  WorkConfig,
  Step,
  Space,
  SpacePreviewResponse,
  Feed,
  FeedState,
  FeedResponse,
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
  DEFAULT_HTTP_METHOD,
  SCRIPT_LANGUAGE_ARGYLL,
  SCRIPT_LANGUAGE_LUA,
  AttributeRole,
  isProducerRole,
  META_KEYS,
  METRIC_SELECTS,
} from "./types";

export type { GoalSets } from "./goals";
export { fromGoalSets, toGoalSets } from "./goals";

export { ArgyllApi } from "./client";
import { ArgyllApi } from "./client";

export const api = new ArgyllApi();
