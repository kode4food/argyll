import { act, renderHook } from "@testing-library/react";
import { FeedState } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { useFeedSubscription } from "./useFeedSubscription";

const watched: FeedState = {
  id: "total",
  version: "v1",
  value: 0,
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

const subscribed = (data: unknown) => ({
  type: "subscribed",
  items: [{ id: ["feed", "total:v1"], data, sequence: 3 }],
});

describe("useFeedSubscription", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useFlowStore.setState({ watchedFeed: null });
  });

  test("does not subscribe without a watched Feed", () => {
    const client = makeClient();
    renderSubscription(client);
    expect(client.subscribe).not.toHaveBeenCalled();
  });

  test("subscribes to the watched Feed version with its state", () => {
    useFlowStore.setState({ watchedFeed: watched });
    const client = makeClient();
    const { unmount } = renderSubscription(client);

    expect(client.subscribe).toHaveBeenCalledWith(
      {
        aggregate_ids: [["feed", "total:v1"]],
        include_state: true,
        event_types: ["feed_folded"],
      },
      expect.any(Function)
    );
    unmount();
    expect(client.unsubscribe).toHaveBeenCalledWith("1");
  });

  test("takes the folded value it subscribed to", () => {
    useFlowStore.setState({ watchedFeed: watched });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(
        subscribed({ ...watched, value: 9, last_updated: "2024-01-03" })
      );
    });

    expect(useFlowStore.getState().watchedFeed).toEqual({
      ...watched,
      value: 9,
      last_updated: "2024-01-03",
    });
  });

  test("keeps the seed when nothing has folded", () => {
    useFlowStore.setState({ watchedFeed: watched });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)(subscribed({ version: "", value: null }));
    });

    expect(useFlowStore.getState().watchedFeed).toEqual(watched);
  });

  test("applies folded values", () => {
    useFlowStore.setState({ watchedFeed: watched });
    const client = makeClient();
    renderSubscription(client);

    act(() => {
      handlerOf(client)({
        type: "feed_folded",
        data: { value: 5 },
        timestamp: Date.UTC(2024, 0, 2),
      });
    });

    expect(useFlowStore.getState().watchedFeed).toEqual({
      ...watched,
      value: 5,
      last_updated: "2024-01-02T00:00:00.000Z",
    });
  });

  test("ignores events once the Feed is no longer watched", () => {
    useFlowStore.setState({ watchedFeed: watched });
    const client = makeClient();
    renderSubscription(client);
    const handler = handlerOf(client);

    act(() => {
      useFlowStore.setState({ watchedFeed: null });
    });
    handler({ type: "feed_folded", data: { value: 1 }, timestamp: 1 });

    expect(useFlowStore.getState().watchedFeed).toBeNull();
  });
});
