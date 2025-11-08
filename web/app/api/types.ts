export type WorkflowStatus =
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
export type HealthStatus = "healthy" | "unhealthy" | "unconfigured" | "unknown";
export type StepType = "sync" | "async" | "script";

export const SCRIPT_LANGUAGE_ALE = "ale";
export const SCRIPT_LANGUAGE_LUA = "lua";

export enum AttributeRole {
  Required = "required",
  Optional = "optional",
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

export interface WorkConfig {
  max_retries?: number;
  backoff_ms?: number;
  max_backoff_ms?: number;
  backoff_type?: "fixed" | "linear" | "exponential";
  parallelism?: number;
}

export interface Step {
  id: string;
  name: string;
  type: StepType;
  attributes: Record<string, AttributeSpec>;
  predicate?: ScriptConfig;
  version: string;
  work_config?: WorkConfig;

  // Type-specific configurations
  http?: HTTPConfig;
  script?: ScriptConfig;
}

export interface ExecutionPlan {
  goal_steps: string[];
  required_inputs: string[];
  steps: Step[];
}

export interface WorkflowContext {
  id: string;
  status: WorkflowStatus;
  state: Record<string, AttributeValue>;
  error_state?: {
    message: string;
    step_id: string;
    timestamp: string;
  };
  execution_plan?: {
    goal_steps: string[];
    required_inputs: string[];
    steps: Step[];
  };
  started_at: string;
  completed_at?: string;
}

export interface WorkState {
  token: string;
  status: StepStatus;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  error?: string;
  retry_count: number;
  next_retry_at?: string;
}

export interface ExecutionResult {
  step_id: string;
  workflow_id: string;
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

export interface WorkflowProjection {
  id: string;
  status: WorkflowStatus;
  execution_plan: {
    goal_steps: string[];
    required_inputs: string[];
    steps: Step[];
  };
  attributes: Record<string, AttributeValue>;
  executions: Record<string, ExecutionInfo>;
  created_at: string;
  completed_at?: string;
  error?: string;
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
