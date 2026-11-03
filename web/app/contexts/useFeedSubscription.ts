import { useCallback, useEffect } from "react";
import { FeedState } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

const FEED_PREFIX = "feed";
const FEED_EVENT_TYPES = ["feed_folded"];

const watchedFeedKey = (state: ReturnType<typeof useFlowStore.getState>) =>
  state.watchedFeed
    ? `${state.watchedFeed.id}:${state.watchedFeed.version}`
    : null;

// Streams the folded value of the watched Feed. The subscription starts from
// the aggregate's state, which holds no version until something folds, so the
// seed the Feed was opened with stands until then
export function useFeedSubscription(socketClient: SocketClient) {
  const feedKey = useFlowStore(watchedFeedKey);
  const setWatchedFeed = useFlowStore((state) => state.setWatchedFeed);

  const handleFeedEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      const current = useFlowStore.getState().watchedFeed;
      if (!current) return;

      if (event.type === "subscribed") {
        const state = (event as WebSocketSubscribed).items[0]?.data as
          FeedState | undefined;
        if (state?.version) {
          setWatchedFeed({
            ...current,
            value: state.value,
            last_updated: state.last_updated,
          });
        }
        return;
      }

      const wsEvent = event as WebSocketEvent;
      setWatchedFeed({
        ...current,
        value: wsEvent.data?.value,
        last_updated: new Date(wsEvent.timestamp || Date.now()).toISOString(),
      });
    },
    [setWatchedFeed]
  );

  useEffect(() => {
    if (!feedKey) return;
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: [[FEED_PREFIX, feedKey]],
        include_state: true,
        event_types: FEED_EVENT_TYPES,
      },
      handleFeedEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [
    feedKey,
    handleFeedEvent,
    socketClient.subscribe,
    socketClient.unsubscribe,
  ]);
}
