import { apiConfig } from "@/constants/common";
import {
  ExecutionPlan,
  QueryFlowsResponse,
  QueryFlowsRequest,
  Space,
  SpacePreviewResponse,
  Step,
} from "./types";

const REQUEST_TIMEOUT_MS = 30000;

export interface StartFlowRequest {
  id: string;
  goalSteps: string[];
  initialState: Record<string, unknown[]>;
  compensate?: boolean;
  spaceId?: string;
}

export interface ExecutionPlanOptions {
  goalSteps: string[];
  initialState?: Record<string, any[]>;
  spaceId?: string;
  signal?: AbortSignal;
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

  async registerSpace(space: Space): Promise<Space> {
    const response = await this.request<{ space: Space }>("/engine/spaces", {
      method: "POST",
      body: JSON.stringify(space),
    });
    return response.space;
  }

  async updateSpace(spaceId: string, space: Space): Promise<Space> {
    const response = await this.request<{ space: Space }>(
      `/engine/spaces/${spaceId}`,
      {
        method: "PUT",
        body: JSON.stringify(space),
      }
    );
    return response.space;
  }

  async previewSpace(space: Space): Promise<SpacePreviewResponse> {
    return this.request("/engine/spaces/preview", {
      method: "POST",
      body: JSON.stringify(space),
    });
  }

  async unregisterSpace(spaceId: string): Promise<void> {
    await this.request(`/engine/spaces/${spaceId}`, { method: "DELETE" });
  }

  async startFlow(request: StartFlowRequest): Promise<unknown> {
    const {
      id,
      goalSteps,
      initialState,
      compensate = false,
      spaceId,
    } = request;
    return this.request("/engine/flows", {
      method: "POST",
      body: JSON.stringify({
        id,
        goals: goalSteps,
        init: initialState,
        compensate,
        ...(spaceId && { space_id: spaceId }),
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
    request: ExecutionPlanOptions
  ): Promise<ExecutionPlan> {
    const { goalSteps, initialState = {}, spaceId, signal } = request;
    return this.request("/engine/plan", {
      method: "POST",
      body: JSON.stringify({
        goals: goalSteps,
        init: initialState,
        ...(spaceId && { space_id: spaceId }),
      }),
      signal,
    });
  }
}
