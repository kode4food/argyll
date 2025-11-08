import React, { useMemo } from "react";
import { Step, AttributeRole } from "../../api";
import { getArgIcon } from "@/utils/argIcons";

interface WorkflowStatsProps {
  steps: Step[];
  executionSequence: string[];
  resolvedAttributes: string[];
}

interface StepStats {
  requiredInputs: number;
  optionalInputs: number;
  outputs: number;
  resolvedRequired: number;
  resolvedOptional: number;
  resolvedOutputs: number;
}

const WorkflowStats: React.FC<WorkflowStatsProps> = React.memo(
  function WorkflowStats({ steps, executionSequence, resolvedAttributes }) {
    const stats: StepStats = useMemo(() => {
      const planStepIds = new Set(executionSequence);
      const planSteps = steps.filter((step) => planStepIds.has(step.id));
      const resolved = new Set(resolvedAttributes);

      return planSteps.reduce(
        (acc, step) => {
          const requiredKeys: string[] = [];
          const optionalKeys: string[] = [];
          const outputKeys: string[] = [];

          Object.entries(step.attributes || {}).forEach(([name, spec]) => {
            if (spec.role === AttributeRole.Required) requiredKeys.push(name);
            else if (spec.role === AttributeRole.Optional)
              optionalKeys.push(name);
            else if (spec.role === AttributeRole.Output) outputKeys.push(name);
          });

          const requiredArgs = requiredKeys.length;
          const optionalArgs = optionalKeys.length;
          const outputArgs = outputKeys.length;

          const resolvedReq = requiredKeys.filter((name) =>
            resolved.has(name)
          ).length;
          const resolvedOpt = optionalKeys.filter((name) =>
            resolved.has(name)
          ).length;
          const resolvedOut = outputKeys.filter((name) =>
            resolved.has(name)
          ).length;

          return {
            requiredInputs: acc.requiredInputs + requiredArgs,
            optionalInputs: acc.optionalInputs + optionalArgs,
            outputs: acc.outputs + outputArgs,
            resolvedRequired: acc.resolvedRequired + resolvedReq,
            resolvedOptional: acc.resolvedOptional + resolvedOpt,
            resolvedOutputs: acc.resolvedOutputs + resolvedOut,
          };
        },
        {
          requiredInputs: 0,
          optionalInputs: 0,
          outputs: 0,
          resolvedRequired: 0,
          resolvedOptional: 0,
          resolvedOutputs: 0,
        }
      );
    }, [steps, executionSequence, resolvedAttributes]);

    const RequiredIcon = getArgIcon("required").Icon;
    const OptionalIcon = getArgIcon("optional").Icon;
    const OutputIcon = getArgIcon("output").Icon;

    return (
      <div className="workflow-stats">
        {stats.requiredInputs > 0 && (
          <span className="status-bubble stat-badge stat-badge--required">
            <RequiredIcon className="stat-badge__icon" />
            {stats.resolvedRequired} of {stats.requiredInputs}
          </span>
        )}
        {stats.optionalInputs > 0 && (
          <span className="status-bubble stat-badge stat-badge--optional">
            <OptionalIcon className="stat-badge__icon" />
            {stats.resolvedOptional} of {stats.optionalInputs}
          </span>
        )}
        {stats.outputs > 0 && (
          <span className="status-bubble stat-badge stat-badge--output">
            <OutputIcon className="stat-badge__icon" />
            {stats.resolvedOutputs} of {stats.outputs}
          </span>
        )}
      </div>
    );
  }
);

export default WorkflowStats;
