import { act, renderHook } from "@testing-library/react";
import { useFlowStore } from "@/app/store/flowStore";
import { useFlowSummarySubscription } from "./useFlowSummarySubscription";

const makeClient = () => {
  let next = 0;
  return {
    subscribe: jest.fn((_sub: unknown, _handler: unknown) => String(next++)),
    unsubscribe: jest.fn(),
  };
};

const started = (flowId: string, metadata?: Record<string, string>) => ({
  type: "flow_started",
  id: ["flow", flowId],
  data: { flow_id: flowId, metadata },
  timestamp: Date.UTC(2026, 0, 2),
});

const flowIds = () => useFlowStore.getState().flows.map(({ id }) => id);

describe("useFlowSummarySubscription", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useFlowStore.setState({ flows: [] });
  });

  test("subscribes to Flows started anywhere", () => {
    const client = makeClient();
    const { unmount } = renderHook(() =>
      useFlowSummarySubscription(client as any, [])
    );

    expect(client.subscribe).toHaveBeenCalledTimes(1);
    expect(client.subscribe.mock.calls[0][0]).toEqual({
      event_types: ["flow_started"],
    });

    unmount();
    expect(client.unsubscribe).toHaveBeenCalledWith("0");
  });

  test("lists new top-level Flows and skips child Flows", () => {
    const client = makeClient();
    renderHook(() => useFlowSummarySubscription(client as any, []));
    const handler = client.subscribe.mock.calls[0][1] as (e: any) => void;

    act(() => {
      handler({ type: "subscribed", items: [] });
      handler(started("from-chat"));
      handler(started("parent:step:tkn", { parent_flow_id: "parent" }));
    });

    expect(flowIds()).toEqual(["from-chat"]);
    expect(useFlowStore.getState().flows[0].status).toBe("active");
  });

  test("follows visible Flows to completion", () => {
    const client = makeClient();
    renderHook(() => useFlowSummarySubscription(client as any, ["f1"]));

    expect(client.subscribe).toHaveBeenCalledTimes(2);
    expect(client.subscribe.mock.calls[1][0]).toMatchObject({
      aggregate_ids: [["flow", "f1"]],
    });
    const handler = client.subscribe.mock.calls[1][1] as (e: any) => void;

    act(() => {
      handler({
        type: "flow_failed",
        id: ["flow", "f1"],
        data: { flow_id: "f1", error: "boom" },
        timestamp: Date.UTC(2026, 0, 2),
      });
    });

    expect(useFlowStore.getState().flows[0]).toMatchObject({
      id: "f1",
      status: "failed",
      error: "boom",
    });
  });
});
