export const STEP_LAYOUT = {
  WIDGET_BASE_HEIGHT: 120,
  SECTION_HEIGHT: 25,
  ARG_LINE_HEIGHT: 20,
  VERTICAL_SPACING: 50,
  HORIZONTAL_SPACING: 400,
  VERTICAL_OFFSET: 400,

  HANDLE_SIZE: 8,
  HANDLE_OFFSET: -4,
  HANDLE_BORDER_WIDTH: 2,

  EDGE_WIDTH: 2,
  DASH_PATTERN: "8,4",

  FIT_VIEW_PADDING: 0.1,
} as const;

export const EDGE_COLORS = {
  REQUIRED: "var(--color-edge-required)" as const,
  OPTIONAL: "var(--color-edge-optional)" as const,
  GRAYED: "var(--color-edge-grayed)" as const,
};
