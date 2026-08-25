import { apiConfig } from "@/constants/common";
import {
  EngineState,
  ExecutionPlan,
  QueryFlowsResponse,
  QueryFlowsRequest,
  Step,
} from "./types";

const REQUEST_TIMEOUT_MS = 30000;

export interface StartFlowRequest {
  id: string;
  goalSteps: string[];
  initialState: Record<string, unknown[]>;
  compensate?: boolean;
}

export class ArgyllApi {
  private readonly baseURL: string;

  constructor(baseURL: string = apiConfig.BASE_URL) {
    this.baseURL = baseURL;
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const timeout = AbortSignal.timeout(REQUEST_TIMEOUT_MS);
    const signal = init.signal
      ? AbortSignal.any([init.signal, timeout])
      : timeout;
    const response = await fetch(this.baseURL + path, {
      ...init,
      signal,
      headers: {
        "Content-Type": "application/json",
        ...init.headers,
      },
    });
    const data = await response.json().catch(() => null);

    if (!response.ok) {
      const message =
        data && typeof data === "object" && "error" in data
          ? String(data.error)
          : `${response.status} ${response.statusText}`;
      throw new Error(message);
    }

    return data as T;
  }

  async registerStep(step: Step): Promise<Step> {
    const response = await this.request<{ step: Step }>("/engine/steps", {
      method: "POST",
      body: JSON.stringify(step),
    });
    return response.step;
  }

  async updateStep(stepId: string, step: Step): Promise<Step> {
    const response = await this.request<{ step: Step }>(
      `/engine/steps/${stepId}`,
      {
        method: "PUT",
        body: JSON.stringify(step),
      }
    );
    return response.step;
  }

  async startFlow(request: StartFlowRequest): Promise<unknown> {
    const { id, goalSteps, initialState, compensate = false } = request;
    return this.request("/engine/flows", {
      method: "POST",
      body: JSON.stringify({
        id,
        goals: goalSteps,
        init: initialState,
        compensate,
      }),
    });
  }

  async queryFlows(request: QueryFlowsRequest): Promise<QueryFlowsResponse> {
    return this.request("/engine/flows/query", {
      method: "POST",
      body: JSON.stringify(request),
    });
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
    return this.request("/engine/plan", {
      method: "POST",
      body: JSON.stringify({
        goals: goalSteps,
        init: initialState,
      }),
      signal,
    });
  }

  async getEngine(): Promise<EngineState> {
    return this.request("/engine");
  }
}
