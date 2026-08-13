import { flowIdGeneration } from "@/constants/common";

export const generatePadded = (): string => {
  return Math.floor(Math.random() * flowIdGeneration.RANDOM_RANGE)
    .toString()
    .padStart(flowIdGeneration.PADDING_LENGTH, "0");
};

export const generateFlowId = (): string => {
  return `${flowIdGeneration.PREFIX}-${generatePadded()}`;
};

export const sanitizeFlowID = (id: string): string => {
  let sanitized = id.toLowerCase();
  sanitized = sanitized.replace(/[^a-z0-9_.\-+ ]/g, "");
  sanitized = sanitized.replace(/ /g, "-");
  sanitized = sanitized.replace(/^-+|-+$/g, "");
  return sanitized;
};
