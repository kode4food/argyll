export type FlowStatus =
  | "pending"
  | "active"
  | "completed"
  | "failed"
  | "stopped";

export type StepStatus =
  | "pending"
  | "active"
  | "completed"
  | "skipped"
  | "failed";
export type WorkStatus =
  | "pending"
  | "active"
  | "succeeded"
  | "failed"
  | "not_completed";
export type HealthStatus = "healthy" | "unhealthy" | "unconfigured" | "unknown";
export type StepType = "sync" | "async" | "script" | "flow";

export const SCRIPT_LANGUAGE_ALE = "ale";
export const SCRIPT_LANGUAGE_JPATH = "jpath";
export const SCRIPT_LANGUAGE_LUA = "lua";

export enum AttributeRole {
  Required = "required",
  Optional = "optional",
  Const = "const",
  Output = "output",
}

export enum AttributeType {
  String = "string",
  Number = "number",
  Boolean = "boolean",
  Object = "object",
  Array = "array",
  Null = "null",
  Any = "any",
}

export interface AttributeSpec {
  role: AttributeRole;
  type?: AttributeType;
  default?: string;
  mapping?: string;
  for_each?: boolean;
}

export interface HTTPConfig {
  endpoint: string;
  health_check?: string;
  timeout: number;
}

export interface ScriptConfig {
  language: string;
  script: string;
}

export interface FlowConfig {
  goals: string[];
  input_map?: Record<string, string>;
  output_map?: Record<string, string>;
}

export interface WorkConfig {
  max_retries?: number;
  backoff?: number;
  max_backoff?: number;
  backoff_type?: "fixed" | "linear" | "exponential";
  parallelism?: number;
}

export interface Step {
  id: string;
  name: string;
  type: StepType;
  attributes: Record<string, AttributeSpec>;
  labels?: Record<string, string>;
  predicate?: ScriptConfig;
  work_config?: WorkConfig;
  memoizable?: boolean;

  // Type-specific configurations
  http?: HTTPConfig;
  script?: ScriptConfig;
  flow?: FlowConfig;
}

export interface Dependencies {
  providers: string[];
  consumers: string[];
}

export interface ExcludedSteps {
  satisfied?: Record<string, string[]>;
  missing?: Record<string, string[]>;
}

export interface ExecutionPlan {
  goals: string[];
  required: string[];
  steps: Record<string, Step>;
  attributes: Record<string, Dependencies>;
  excluded?: ExcludedSteps;
}

export interface FlowContext {
  id: string;
  status: FlowStatus;
  state: Record<string, AttributeValue>;
  error_state?: {
    message: string;
    step_id: string;
    timestamp: string;
  };
  plan?: ExecutionPlan;
  started_at: string;
  completed_at?: string;
}

export interface WorkState {
  token: string;
  status: WorkStatus;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  error?: string;
  retry_count: number;
  next_retry_at?: string;
}

export interface ExecutionResult {
  step_id: string;
  flow_id: string;
  status: StepStatus;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  error_message?: string;
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  work_items?: Record<string, WorkState>;
}

export interface StepHealth {
  status: HealthStatus;
  error?: string;
}

export interface AttributeValue {
  value: any;
  step?: string;
}

export interface FlowProjection {
  id: string;
  status: FlowStatus;
  plan: ExecutionPlan;
  attributes: Record<string, AttributeValue>;
  executions: Record<string, ExecutionInfo>;
  created_at: string;
  completed_at?: string;
  error?: string;
}

export interface QueryFlowsItem {
  id: string;
  digest?: {
    status: FlowStatus;
    created_at: string;
    completed_at?: string;
    labels?: Record<string, string>;
    error?: string;
  };
}

export interface QueryFlowsResponse {
  flows: QueryFlowsItem[];
  count: number;
  total?: number;
  has_more?: boolean;
  next_cursor?: string;
}

export type FlowSort = "recent_desc" | "recent_asc";

export interface QueryFlowsRequest {
  id_prefix?: string;
  labels?: Record<string, string>;
  statuses?: FlowStatus[];
  limit?: number;
  cursor?: string;
  sort?: FlowSort;
}

export interface WorkItemState {
  status: "pending" | "active" | "completed" | "failed";
  started_at: string;
  completed_at?: string;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  error?: string;
}

export interface ExecutionInfo {
  status: StepStatus;
  started_at: string;
  completed_at?: string;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  duration?: number;
  error?: string;
  work_items?: Record<string, WorkItemState>;
}
