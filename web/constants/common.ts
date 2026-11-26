// The engine, chat, and sign-in share the page's origin
export const API_CONFIG = {
  WS_URL: `${window.location.origin.replace(/^http/, "ws")}/engine/ws`,
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
  PROTOCOL: "argyll",
  TOKEN_PREFIX: "bearer.",
} as const;

// UI Constants
export const UI = {
  TOOLTIP_MIN_WIDTH: 300,
  TOOLTIP_MAX_WIDTH: 400,
  TOOLTIP_OFFSET: 10,
} as const;
