import axios, { AxiosInstance } from "axios";
import { API_CONFIG } from "@/constants/common";
import {
  EngineState,
  ExecutionPlan,
  QueryFlowsResponse,
  QueryFlowsRequest,
  Step,
} from "./types";

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
    initialState: Record<string, any[]>
  ): Promise<any> {
    const response = await this.client.post("/engine/flow", {
      id,
      goals: goalSteps,
      init: initialState,
    });
    return response.data;
  }

  async queryFlows(request: QueryFlowsRequest): Promise<QueryFlowsResponse> {
    const response = await this.client.post("/engine/flow/query", request);
    return response.data;
  }

  async listFlowsPage(opts?: {
    limit?: number;
    cursor?: string;
  }): Promise<QueryFlowsResponse> {
    return this.queryFlows({
      limit: opts?.limit,
      cursor: opts?.cursor,
      sort: "recent_desc",
    });
  }

  async getExecutionPlan(
    goalSteps: string[],
    initialState: Record<string, any[]> = {},
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

  async getEngine(): Promise<EngineState> {
    const response = await this.client.get("/engine");
    return response.data;
  }
}
