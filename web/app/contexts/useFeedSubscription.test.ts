import { act, renderHook } from "@testing-library/react";
import { FeedState } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { useFeedSubscription } from "./useFeedSubscription";

const total: FeedState = {
  id: "total",
  version: "v1",
  value: 0,
  last_updated: "2024-01-01T00:00:00.000Z",
};

const count: FeedState = {
  id: "count",
  version: "v2",
  value: 1,
  last_updated: "2024-01-01T00:00:00.000Z",
};

const makeClient = () => ({
  subscribe: jest.fn((_sub: unknown, _handler: unknown) => "1"),
  unsubscribe: jest.fn(),
});

const renderSubscription = (client: ReturnType<typeof makeClient>) =>
  renderHook(() => useFeedSubscription(client as any));

const handlerOf = (client: ReturnType<typeof makeClient>) =>
  client.subscribe.mock.calls[0][1] as (event: any) => void;

const subscribed = (key: string, data: unknown) => ({
  type: "subscribed",
  items: [{ id: ["feed", key], data, sequence: 3 }],
});

const folded = (key: string, value: unknown) => ({
  type: "feed_folded",
  id: ["feed", key],
  data: { value },
  timestamp: Date.UTC(2024, 0, 2),
});

const watched = () => useFlowStore.getState().watchedFeeds;

describe("useFeedSubscription", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useFlowStore.setState({ watchedFeeds: {} });
  });

  test("does not subscribe without watched Feeds", () => {
    const client = makeClient();
    renderSubscription(client);
    expect(client.subscribe).not.toHaveBeenCalled();
  });

  test("subscribes to every watched Feed version with its state", () => {
    useFlowStore.setState({ watchedFeeds: { total, count } });
    const client = makeClient();
    const { unmount } = renderSubscription(client);

    expect(client.subscribe).toHaveBeenCalledWith(
      {
        aggregate_ids: [
          ["feed", "count:v2"],
          ["feed", "total:v1"],
        ],
        include_state: true,
        event_types: ["feed_folded"],
      },
      expect.any(Function)
    );
    unmount();
    expect(client.unsubscribe).toHaveBeenCalledWith("1");
  });

  test("keeps one subscription while values change", () => {
    useFlowStore.setState({ watchedFeeds: { total } });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(folded("total:v1", 5));
    });

    expect(client.subscribe).toHaveBeenCalledTimes(1);
    expect(client.unsubscribe).not.toHaveBeenCalled();
  });

  test("resubscribes when a version changes", () => {
    useFlowStore.setState({ watchedFeeds: { total } });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      useFlowStore.setState({
        watchedFeeds: { total: { ...total, version: "v2" } },
      });
    });

    expect(client.unsubscribe).toHaveBeenCalledWith("1");
    expect(client.subscribe.mock.calls[1][0]).toMatchObject({
      aggregate_ids: [["feed", "total:v2"]],
    });
  });

  test("takes the folded value it subscribed to", () => {
    useFlowStore.setState({ watchedFeeds: { total, count } });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(
        subscribed("total:v1", {
          ...total,
          value: 9,
          last_updated: "2024-01-03",
        })
      );
    });

    expect(watched()).toEqual({
      total: { ...total, value: 9, last_updated: "2024-01-03" },
      count,
    });
  });

  test("keeps the seed when nothing has folded", () => {
    useFlowStore.setState({ watchedFeeds: { total } });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(subscribed("total:v1", { version: "", value: null }));
    });

    expect(watched()).toEqual({ total });
  });

  test("applies folded values to the matching Feed", () => {
    useFlowStore.setState({ watchedFeeds: { total, count } });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(folded("count:v2", 5));
    });

    expect(watched()).toEqual({
      total,
      count: { ...count, value: 5, last_updated: "2024-01-02T00:00:00.000Z" },
    });
  });

  test("ignores events once a Feed is no longer watched", () => {
    useFlowStore.setState({ watchedFeeds: { total } });
    const client = makeClient();
    renderSubscription(client);
    const handler = handlerOf(client);

    act(() => {
      useFlowStore.setState({ watchedFeeds: {} });
    });
    handler(folded("total:v1", 1));
    handler(subscribed("total:v1", { ...total, value: 2 }));

    expect(watched()).toEqual({});
  });
});
