import { useCallback, useEffect } from "react";
import { FeedState } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

const FEED_PREFIX = "feed";
const FEED_EVENT_TYPES = ["feed_folded"];

const feedKey = (feed: FeedState) => `${feed.id}:${feed.version}`;

// Serialized so the selector result compares by value across store updates
const watchedFeedKeys = (state: ReturnType<typeof useFlowStore.getState>) =>
  JSON.stringify(Object.values(state.watchedFeeds).map(feedKey).sort());

const watchedFeedFor = (aggregateId: string[] | undefined) =>
  Object.values(useFlowStore.getState().watchedFeeds).find(
    (feed) => feedKey(feed) === aggregateId?.[1]
  );

// Streams the folded values of the watched Feeds. A subscription starts from
// each aggregate's state, which holds no version until something folds, so
// the seed a Feed was opened with stands until then
export function useFeedSubscription(socketClient: SocketClient) {
  const keys = useFlowStore(watchedFeedKeys);
  const setWatchedFeed = useFlowStore((state) => state.setWatchedFeed);

  const handleFeedEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        (event as WebSocketSubscribed).items.forEach((item) => {
          const current = watchedFeedFor(item.id);
          const state = item.data as FeedState | undefined;
          if (!current || !state?.version) return;
          setWatchedFeed({
            ...current,
            value: state.value,
            last_updated: state.last_updated,
          });
        });
        return;
      }

      const wsEvent = event as WebSocketEvent;
      const current = watchedFeedFor(wsEvent.id);
      if (!current) return;
      setWatchedFeed({
        ...current,
        value: wsEvent.data?.value,
        last_updated: new Date(wsEvent.timestamp || Date.now()).toISOString(),
      });
    },
    [setWatchedFeed]
  );

  useEffect(() => {
    const aggregateKeys: string[] = JSON.parse(keys);
    if (aggregateKeys.length === 0) return;
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: aggregateKeys.map((key) => [FEED_PREFIX, key]),
        include_state: true,
        event_types: FEED_EVENT_TYPES,
      },
      handleFeedEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [keys, handleFeedEvent, socketClient.subscribe, socketClient.unsubscribe]);
}
