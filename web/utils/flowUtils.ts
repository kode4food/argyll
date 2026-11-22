import { FLOW_ID_GENERATION } from "@/constants/common";

export const generatePadded = (): string => {
  return Math.floor(Math.random() * FLOW_ID_GENERATION.RANDOM_RANGE)
    .toString()
    .padStart(FLOW_ID_GENERATION.PADDING_LENGTH, "0");
};

export const generateFlowId = (): string => {
  return `${FLOW_ID_GENERATION.PREFIX}-${generatePadded()}`;
};

export const sanitizeFlowID = (id: string): string => {
  let sanitized = id.toLowerCase();
  sanitized = sanitized.replace(/[^a-z0-9_.\-+ ]/g, "");
  sanitized = sanitized.replace(/ /g, "-");
  sanitized = sanitized.replace(/^-+|-+$/g, "");
  return sanitized;
};
