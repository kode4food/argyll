import axios, { AxiosInstance } from "axios";
import { API_CONFIG } from "@/constants/common";
import { ExecutionPlan, FlowContext, FlowsListItem, Step } from "./types";

export class ArgyllApi {
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

  private convertListItem(item: FlowsListItem): FlowContext {
    return {
      id: item.id,
      status: item.digest?.status || "active",
      state: {},
      error_state: item.digest?.error
        ? {
            message: item.digest.error,
            step_id: "",
            timestamp: new Date().toISOString(),
          }
        : undefined,
      plan: undefined,
      started_at: item.digest?.created_at || new Date().toISOString(),
      completed_at: item.digest?.completed_at,
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
    goalSteps: string[],
    initialState: Record<string, any>
  ): Promise<any> {
    const response = await this.client.post("/engine/flow", {
      id,
      goals: goalSteps,
      init: initialState,
    });
    return response.data;
  }

  async listFlows(): Promise<FlowContext[]> {
    const response = await this.client.get("/engine/flow");
    const items: FlowsListItem[] = response.data.flows || [];
    return items.map((item) => this.convertListItem(item));
  }

  async getExecutionPlan(
    goalSteps: string[],
    initialState: Record<string, any> = {},
    signal?: AbortSignal
  ): Promise<ExecutionPlan> {
    const response = await this.client.post(
      "/engine/plan",
      {
        goals: goalSteps,
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
