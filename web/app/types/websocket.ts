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
  sub_id?: string;
}

export interface WebSocketSubscribe {
  sub_id?: string;
  aggregate_id?: string[];
  event_types?: string[];
}

export interface WebSocketUnsubscribe {
  sub_id: string;
}

export interface WebSocketSubscribed {
  type: "subscribed";
  id: string[];
  data: unknown;
  sequence: number;
  sub_id?: string;
}
