import { Step } from "@/app/api";
import { Label } from "./stepEditorTypes";

export function buildLabelRows(labels?: Record<string, string>): Label[] {
  if (!labels) return [];
  const timestamp = Date.now();
  return Object.keys(labels)
    .sort((a, b) => a.localeCompare(b))
    .map((key, index) => ({
      id: `label-${index}-${timestamp}`,
      key,
      value: labels[key],
    }));
}

export function buildLabelsFromStep(step: Step | null): Label[] {
  return buildLabelRows(step?.labels);
}

export function createStepLabels(
  labels: Label[]
): Record<string, string> | undefined {
  const res: Record<string, string> = {};
  labels.forEach(({ key, value }) => {
    const name = key.trim();
    if (name) res[name] = value.trim();
  });
  return Object.keys(res).length > 0 ? res : undefined;
}
