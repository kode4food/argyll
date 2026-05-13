import { useMemo } from "react";
import { AttributeRole, Step, FlowContext, ExecutionResult } from "@/app/api";
import { buildProvenanceMap } from "@/utils/stepNodeUtils";

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
    if (!execution || execution.status === "pending") {
      return new Set<string>();
    }

    const unsatisfied = new Set(execution.unsatisfied || []);
    return new Set(
      Object.entries(step.attributes)
        .filter(
          ([name, spec]) =>
            (spec.role === AttributeRole.Required ||
              spec.role === AttributeRole.Optional) &&
            !unsatisfied.has(name)
        )
        .map(([name]) => name)
    );
  }, [execution?.status, execution?.unsatisfied, step.attributes]);

  return {
    execution,
    resolved,
    provenance,
    satisfied,
  };
};
