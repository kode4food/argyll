import { WORKFLOW_ID_GENERATION } from "@/constants/common";

export const generatePadded = (): string => {
  return Math.floor(Math.random() * WORKFLOW_ID_GENERATION.RANDOM_RANGE)
    .toString()
    .padStart(WORKFLOW_ID_GENERATION.PADDING_LENGTH, "0");
};

export const generateWorkflowId = (): string => {
  return `${WORKFLOW_ID_GENERATION.PREFIX}-${generatePadded()}`;
};

export const sanitizeWorkflowID = (id: string): string => {
  let sanitized = id.toLowerCase();
  sanitized = sanitized.replace(/[^a-z0-9_.\-+ ]/g, "");
  sanitized = sanitized.replace(/ /g, "-");
  sanitized = sanitized.replace(/^-+|-+$/g, "");
  return sanitized;
};
