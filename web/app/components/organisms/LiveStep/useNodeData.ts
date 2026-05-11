import { useMemo } from "react";
import { Step, FlowContext, ExecutionResult } from "@/app/api";
import {
  buildProvenanceMap,
  calculateSatisfiedArgs,
} from "@/utils/stepNodeUtils";

export interface NodeDataResult {
  execution: ExecutionResult | undefined;
  resolved: Set<string>;
  provenance: Map<string, string>;
  satisfied: Set<string>;
}

export interface FlowExecutionData {
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
}

export const useNodeData = (
  step: Step,
  flowExecution: FlowExecutionData = {}
): NodeDataResult => {
  const { flowData, executions = [], resolvedAttributes = [] } = flowExecution;
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
