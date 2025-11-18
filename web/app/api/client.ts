import axios, { AxiosInstance } from "axios";
import { API_CONFIG } from "@/constants/common";
import {
  Step,
  FlowContext,
  ExecutionResult,
  ExecutionPlan,
  FlowProjection,
  StepStatus,
} from "./types";

export class SpudsApi {
  private client: AxiosInstance;

  constructor(baseURL: string = API_CONFIG.BASE_URL) {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        "Content-Type": "application/json",
      },
    });
  }

  private convertProjection(projection: FlowProjection): FlowContext {
    let errorState = undefined;
    if (projection.error) {
      errorState = {
        message: projection.error,
        step_id: "",
        timestamp: new Date().toISOString(),
      };
    }

    let executionPlan = undefined;
    if (projection.plan && Object.keys(projection.plan.steps).length > 0) {
      executionPlan = projection.plan;
    }

    return {
      id: projection.id,
      status: projection.status,
      state: projection.attributes || {},
      error_state: errorState,
      plan: executionPlan,
      started_at: projection.created_at,
      completed_at: projection.completed_at,
    };
  }

  async registerStep(step: Step): Promise<Step> {
    const response = await this.client.post("/engine/step", step);
    return response.data.step;
  }

  async updateStep(stepId: string, step: Step): Promise<Step> {
    const response = await this.client.put(`/engine/step/${stepId}`, step);
    return response.data.step;
  }

  async startFlow(
    id: string,
    goalStepIds: string[],
    initialState: Record<string, any>
  ): Promise<any> {
    const response = await this.client.post("/engine/flow", {
      id,
      goals: goalStepIds,
      init: initialState,
    });
    return response.data;
  }

  async getFlowWithEvents(id: string): Promise<{
    flow: FlowContext;
    executions: ExecutionResult[];
  }> {
    const response = await this.client.get(`/engine/flow/${id}`);
    const projection: FlowProjection = response.data;

    const flow = this.convertProjection(projection);
    const executions = this.extractExecutions(projection, id);

    return {
      flow,
      executions,
    };
  }

  async listFlows(): Promise<FlowContext[]> {
    const response = await this.client.get("/engine/flow");
    const projections: FlowProjection[] = response.data.flows || [];
    return projections.map((p) => this.convertProjection(p));
  }

  async getExecutions(flowId: string): Promise<ExecutionResult[]> {
    const response = await this.client.get(`/engine/flow/${flowId}`);
    const projection: FlowProjection = response.data;

    return this.extractExecutions(projection, flowId);
  }

  private extractExecutions(
    projection: FlowProjection,
    flowId: string
  ): ExecutionResult[] {
    if (!projection.executions) {
      return [];
    }

    const executionStatusMap: Record<string, StepStatus> = {
      pending: "pending",
      active: "active",
      completed: "completed",
      failed: "failed",
      skipped: "skipped",
    };

    return Object.entries(projection.executions).map(([stepId, exec]) => ({
      step_id: stepId,
      flow_id: flowId,
      status: executionStatusMap[exec.status] || "pending",
      inputs: exec.inputs,
      outputs: exec.outputs,
      error_message: exec.error,
      started_at: exec.started_at,
      completed_at: exec.completed_at,
      duration_ms: exec.duration,
    }));
  }

  async getExecutionPlan(
    goalStepIds: string[],
    initialState: Record<string, any> = {},
    signal?: AbortSignal
  ): Promise<ExecutionPlan> {
    const response = await this.client.post(
      "/engine/plan",
      {
        goals: goalStepIds,
        init: initialState,
      },
      {
        signal,
      }
    );
    return response.data;
  }

  async getEngineState(): Promise<{
    steps: Record<string, Step>;
    health: Record<string, any>;
  }> {
    const response = await this.client.get("/engine");
    return response.data;
  }
}
