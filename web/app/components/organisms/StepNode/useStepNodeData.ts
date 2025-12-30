import { useMemo } from "react";
import { Step, FlowContext, ExecutionResult } from "@/app/api";
import { buildProvenanceMap, calculateSatisfiedArgs } from "./stepNodeUtils";

export interface StepNodeDataResult {
  execution: ExecutionResult | undefined;
  resolved: Set<string>;
  provenance: Map<string, string>;
  satisfied: Set<string>;
}

/**
 * Hook that computes derived data for a step node
 * Handles memoization of expensive calculations to prevent unnecessary re-renders
 *
 * @param step - The step data
 * @param flowData - Flow context with state information
 * @param executions - List of execution results
 * @param resolvedAttributes - List of resolved attribute names
 * @returns Object containing execution, resolved, provenance, and satisfied data
 */
export const useStepNodeData = (
  step: Step,
  flowData: FlowContext | null | undefined,
  executions: ExecutionResult[] = [],
  resolvedAttributes: string[] = []
): StepNodeDataResult => {
  const execution = useMemo(
    () => executions.find((exec) => exec.step_id === step.id),
    [executions, step.id]
  );

  const resolved = useMemo(
    () => new Set(resolvedAttributes),
    [resolvedAttributes]
  );

  const provenance = useMemo(() => {
    return buildProvenanceMap(flowData?.state);
  }, [flowData?.state]);

  const satisfied = useMemo(() => {
    return calculateSatisfiedArgs(step.attributes || {}, resolved);
  }, [step.attributes, resolved]);

  return {
    execution,
    resolved,
    provenance,
    satisfied,
  };
};
