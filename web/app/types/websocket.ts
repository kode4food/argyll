export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "reconnecting"
  | "failed";

export interface WebSocketEvent {
  type: string;
  data: any;
  timestamp: number;
  sequence: number;
  id: string[];
}

export interface WebSocketSubscription {
  aggregate_id?: string[];
  event_types?: string[];
}

export interface WebSocketSubscribeState {
  type: "subscribe_state";
  id: string[];
  data: unknown;
  sequence: number;
}
