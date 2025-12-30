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
  from_sequence?: number;
}
