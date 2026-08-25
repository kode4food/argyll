import { ArgyllApi } from "./client";
import { AttributeRole, AttributeType, Step } from "./types";

const fetchMock = jest.fn() as jest.MockedFunction<typeof fetch>;
global.fetch = fetchMock;

const respond = (data: unknown, init: Partial<Response> = {}) => {
  fetchMock.mockResolvedValueOnce({
    ok: true,
    status: 200,
    statusText: "OK",
    json: jest.fn().mockResolvedValue(data),
    ...init,
  } as Response);
};

const step: Step = {
  id: "step-1",
  name: "Test Step",
  type: "sync",
  attributes: {},
  http: { endpoint: "http://localhost:8080/test", timeout: 5000 },
};

describe("ArgyllApi", () => {
  const api = new ArgyllApi("http://localhost:8080");

  beforeEach(() => {
    fetchMock.mockReset();
  });

  test.each([
    ["registers", () => api.registerStep(step), "POST", "/engine/steps"],
    [
      "updates",
      () => api.updateStep(step.id, step),
      "PUT",
      `/engine/steps/${step.id}`,
    ],
  ])("%s a step", async (_, request, method, path) => {
    respond({ step });

    await expect(request()).resolves.toEqual(step);
    expect(fetchMock).toHaveBeenCalledWith(
      `http://localhost:8080${path}`,
      expect.objectContaining({ method, body: JSON.stringify(step) })
    );
  });

  test("starts a flow", async () => {
    const response = { flow_id: "wf-1" };
    respond(response);

    await expect(
      api.startFlow({
        id: "wf-1",
        goalSteps: ["step-1"],
        initialState: { input: ["value"] },
        compensate: true,
      })
    ).resolves.toEqual(response);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/engine/flows",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          id: "wf-1",
          goals: ["step-1"],
          init: { input: ["value"] },
          compensate: true,
        }),
      })
    );
  });

  test("queries recent flows", async () => {
    respond({ flows: [] });

    await expect(api.listFlowsPage()).resolves.toEqual({ flows: [] });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/engine/flows/query",
      expect.objectContaining({
        body: JSON.stringify({ sort: "recent_desc" }),
      })
    );
  });

  test("fetches an execution plan with the caller signal", async () => {
    const plan = {
      goals: ["step-2"],
      required: ["input1"],
      steps: {
        "step-1": {
          step: {
            ...step,
            attributes: {
              input1: {
                role: AttributeRole.Output,
                type: AttributeType.String,
              },
            },
          },
        },
      },
      attributes: {},
    };
    const controller = new AbortController();
    respond(plan);

    await expect(
      api.getExecutionPlan(["step-2"], { input: ["value"] }, controller.signal)
    ).resolves.toEqual(plan);
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.body).toBe(
      JSON.stringify({ goals: ["step-2"], init: { input: ["value"] } })
    );
    expect(init.signal).toBeInstanceOf(AbortSignal);
  });

  test("fetches engine state", async () => {
    const state = { steps: { "step-1": step }, health: {} };
    respond(state);

    await expect(api.getEngine()).resolves.toEqual(state);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/engine",
      expect.objectContaining({
        headers: { "Content-Type": "application/json" },
      })
    );
  });

  test("throws the API error message", async () => {
    respond(
      { error: "Server error" },
      { ok: false, status: 400, statusText: "Bad Request" }
    );

    await expect(api.getEngine()).rejects.toThrow("Server error");
  });
});
