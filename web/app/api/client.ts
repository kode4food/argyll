import { API_CONFIG } from "@/constants/common";
import { useAuthStore } from "@/app/store/authStore";
import {
  ExecutionPlan,
  Feed,
  FeedResponse,
  AgentCapabilities,
  AgentChatReply,
  AgentChatRequest,
  QueryFlowsResponse,
  QueryFlowsRequest,
  Space,
  SpacePreviewResponse,
  Step,
} from "./types";
import { fromGoalSets, GoalSets } from "./goals";

const REQUEST_TIMEOUT_MS = 30000;
const HTTP_UNAUTHORIZED = 401;
const CHAT_TIMEOUT_MS = 180000;

export interface StartFlowRequest {
  id: string;
  goalSteps: GoalSets;
  initialState: Record<string, unknown[]>;
  spaceId?: string;
}

export interface ExecutionPlanOptions {
  goalSteps: GoalSets;
  initialState?: Record<string, any[]>;
  spaceId?: string;
  signal?: AbortSignal;
}

export class ArgyllApi {
  private readonly baseURL: string;

  constructor(baseURL: string = API_CONFIG.BASE_URL) {
    this.baseURL = baseURL;
  }

  private async request<T>(
    path: string,
    init: RequestInit = {},
    timeoutMs = REQUEST_TIMEOUT_MS,
    baseURL = this.baseURL
  ): Promise<T> {
    const timeout = AbortSignal.timeout(timeoutMs);
    const signal = init.signal
      ? AbortSignal.any([init.signal, timeout])
      : timeout;
    const { token, isEnabled, login } = useAuthStore.getState();
    const response = await fetch(baseURL + path, {
      ...init,
      signal,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init.headers,
      },
    });
    const data = await response.json().catch(() => null);

    if (response.status === HTTP_UNAUTHORIZED && isEnabled) {
      void login().catch((error: unknown) =>
        console.error("Sign-in failed", error)
      );
    }
    if (!response.ok) {
      const hasError = data && typeof data === "object" && "error" in data;
      const hasDetail = data && typeof data === "object" && "detail" in data;
      const message = hasError
        ? String(data.error)
        : hasDetail
          ? String(data.detail)
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
    const response = await this.request<SpacePreviewResponse>(
      "/engine/spaces/preview",
      {
        method: "POST",
        body: JSON.stringify(space),
      }
    );
    return { ...response, step_ids: response.step_ids ?? [] };
  }

  async unregisterSpace(spaceId: string): Promise<void> {
    await this.request(`/engine/spaces/${spaceId}`, { method: "DELETE" });
  }

  async listAgentCapabilities(
    signal?: AbortSignal
  ): Promise<AgentCapabilities> {
    const response = await this.request<AgentCapabilities>(
      "/engine/agents/capabilities",
      { signal }
    );
    return { ...response, providers: response.providers ?? [] };
  }

  async chatWithAgent(
    request: AgentChatRequest,
    signal?: AbortSignal
  ): Promise<AgentChatReply> {
    // Chat is its own service, so the engine has no part in it
    return this.request<AgentChatReply>(
      "/chat",
      { method: "POST", body: JSON.stringify(request), signal },
      CHAT_TIMEOUT_MS,
      API_CONFIG.CHAT_URL
    );
  }

  async getFeed(feedId: string): Promise<FeedResponse> {
    return this.request(`/engine/feeds/${feedId}`);
  }

  async registerFeed(feed: Feed): Promise<Feed> {
    const response = await this.request<{ feed: Feed }>("/engine/feeds", {
      method: "POST",
      body: JSON.stringify(feed),
    });
    return response.feed;
  }

  async updateFeed(feedId: string, feed: Feed): Promise<Feed> {
    const response = await this.request<{ feed: Feed }>(
      `/engine/feeds/${feedId}`,
      {
        method: "PUT",
        body: JSON.stringify(feed),
      }
    );
    return response.feed;
  }

  async unregisterFeed(feedId: string): Promise<void> {
    await this.request(`/engine/feeds/${feedId}`, { method: "DELETE" });
  }

  async startFlow(request: StartFlowRequest): Promise<unknown> {
    const { id, goalSteps, initialState, spaceId } = request;
    return this.request("/engine/flows", {
      method: "POST",
      body: JSON.stringify({
        id,
        goals: fromGoalSets(goalSteps),
        init: initialState,
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
    });
  }

  async getExecutionPlan(
    request: ExecutionPlanOptions
  ): Promise<ExecutionPlan> {
    const { goalSteps, initialState = {}, spaceId, signal } = request;
    return this.request("/engine/plan", {
      method: "POST",
      body: JSON.stringify({
        goals: fromGoalSets(goalSteps),
        init: initialState,
        ...(spaceId && { space_id: spaceId }),
      }),
      signal,
    });
  }
}
