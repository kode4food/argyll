import { useCallback, useRef, RefObject } from "react";
import {
  WebSocketEvent,
  WebSocketSubscribe,
  WebSocketSubscribed,
} from "@/app/types/websocket";

export interface SubscriptionEntry {
  onEvent?: (event: WebSocketEvent | WebSocketSubscribed) => void;
  subscription: WebSocketSubscribe;
}

export const sendSubscribeMessage = (
  ws: WebSocket,
  subscription: WebSocketSubscribe
) => {
  ws.send(JSON.stringify({ type: "subscribe", data: subscription }));
};

export const sendUnsubscribeMessage = (ws: WebSocket, sub_id: string) => {
  ws.send(JSON.stringify({ type: "unsubscribe", data: { sub_id } }));
};

export function useSubscriptions(wsRef: RefObject<WebSocket | null>) {
  const nextSubscriptionIdRef = useRef(0);
  const subscriptionsRef = useRef<Map<string, SubscriptionEntry>>(new Map());

  const subscribe = useCallback(
    (
      subscription: WebSocketSubscribe,
      handler?: (event: WebSocketEvent | WebSocketSubscribed) => void
    ) => {
      const subscriptionId = String(nextSubscriptionIdRef.current);
      nextSubscriptionIdRef.current += 1;

      const nextSubscription = { ...subscription, sub_id: subscriptionId };
      subscriptionsRef.current.set(subscriptionId, {
        subscription: nextSubscription,
        onEvent: handler,
      });

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        sendSubscribeMessage(wsRef.current, nextSubscription);
      }

      return subscriptionId;
    },
    [wsRef]
  );

  const unsubscribe = useCallback(
    (subscriptionId: string) => {
      const hadSubscription = subscriptionsRef.current.delete(subscriptionId);
      if (!hadSubscription || wsRef.current?.readyState !== WebSocket.OPEN) {
        return;
      }
      sendUnsubscribeMessage(wsRef.current, subscriptionId);
    },
    [wsRef]
  );

  return { subscribe, unsubscribe, subscriptionsRef };
}
