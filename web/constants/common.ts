// API Configuration
export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL!,
  WS_URL: process.env.NEXT_PUBLIC_WS_URL!,
} as const;

// Common constants
export const FLOW_ID_GENERATION = {
  RANDOM_RANGE: 100000000,
  PADDING_LENGTH: 8,
  PREFIX: "flow",
} as const;

// WebSocket
export const WEBSOCKET = {
  INITIAL_RECONNECT_DELAY: 1000,
  MAX_RECONNECT_DELAY: 30000,
  RECONNECT_MULTIPLIER: 1.5,
  MAX_RECONNECT_ATTEMPTS: 10,
  BUFFER_SIZE: 256,
  HEARTBEAT_INTERVAL: 30000,
} as const;

// UI Constants
export const UI = {
  TOOLTIP_MIN_WIDTH: 300,
  TOOLTIP_MAX_WIDTH: 400,
  TOOLTIP_OFFSET: 10,
} as const;
