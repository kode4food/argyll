import React from "react";
import {
  CheckCircle2,
  XCircle,
  Award,
  Ban,
  CircleDashed,
  CheckCircle,
  CircleDot,
  CircleSlash,
} from "lucide-react";
import { Step, ExecutionResult, AttributeSpec, AttributeRole } from "../../api";
import Tooltip from "../atoms/Tooltip";
import TooltipSection from "../atoms/TooltipSection";
import { getArgIcon } from "@/utils/argIcons";
import { getSortedAttributes } from "@/utils/stepUtils";
import styles from "./StepAttributesSection.module.css";

interface StepAttributesSectionProps {
  step: Step;
  satisfiedArgs: Set<string>;
  showStatus?: boolean;
  execution?: ExecutionResult;
  attributeProvenance?: Map<string, string>; // attribute name -> step ID that produced it
}

const renderStatusBadge = (
  argType: "required" | "optional" | "output",
  context: {
    isSatisfied: boolean;
    executionStatus?: string;
    isWinner?: boolean;
    isProvidedByUpstream?: boolean;
    wasDefaulted?: boolean;
  }
): React.ReactElement | null => {
  const {
    isSatisfied,
    executionStatus,
    isWinner,
    isProvidedByUpstream,
    wasDefaulted,
  } = context;

  if (argType === "optional" && executionStatus) {
    if (executionStatus === "skipped") {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
        >
          <CircleSlash className={styles.statusIcon} />
        </div>
      );
    }
    if (isProvidedByUpstream) {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
        >
          <CheckCircle className={styles.statusIcon} />
        </div>
      );
    }
    if (wasDefaulted) {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.defaulted} arg-status-badge defaulted`}
        >
          <CircleDot className={styles.statusIcon} />
        </div>
      );
    }
    return (
      <div
        className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
      >
        <CircleDashed className={styles.statusIcon} />
      </div>
    );
  }

  if (argType === "required") {
    if (isSatisfied) {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
        >
          <CheckCircle2 className={styles.statusIcon} />
        </div>
      );
    }
    if (executionStatus === "failed" || executionStatus === "skipped") {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.failed} arg-status-badge failed`}
        >
          <XCircle className={styles.statusIcon} />
        </div>
      );
    }
    return (
      <div
        className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
      >
        <CircleDashed className={styles.statusIcon} />
      </div>
    );
  }

  if (argType === "output") {
    if (executionStatus === "skipped" || executionStatus === "failed") {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
        >
          <Ban className={styles.statusIcon} />
        </div>
      );
    }
    if (executionStatus === "active") {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
        >
          <CircleDashed className={styles.statusIcon} />
        </div>
      );
    }
    if (isWinner) {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
        >
          <Award className={styles.statusIcon} />
        </div>
      );
    }
    if (executionStatus === "completed") {
      return (
        <div
          className={`${styles.argStatusBadge} ${styles.notWinner} arg-status-badge not-winner`}
        >
          <XCircle className={styles.statusIcon} />
        </div>
      );
    }
    return (
      <div
        className={`${styles.argStatusBadge} ${styles.placeholder} arg-status-badge placeholder`}
      />
    );
  }

  return null;
};

interface UnifiedArg {
  name: string;
  type: string;
  argType: "required" | "optional" | "output";
  spec: AttributeSpec;
}

interface ArgValueResult {
  hasValue: boolean;
  value: any;
}

const formatValue = (val: any): string => {
  if (val === null) return "null";
  if (val === undefined) return "undefined";
  if (typeof val === "string") return `"${val}"`;
  if (typeof val === "object") {
    try {
      return JSON.stringify(val, null, 2);
    } catch {
      return String(val);
    }
  }
  return String(val);
};

const getTooltipTitle = (
  argType: "required" | "optional" | "output",
  wasDefaulted?: boolean
) => {
  switch (argType) {
    case "required":
      return "Input Value";
    case "optional":
      return wasDefaulted ? "Default Value" : "Input Value";
    case "output":
      return "Output Value";
  }
};

const getArgValue = (
  arg: UnifiedArg,
  execution?: ExecutionResult
): ArgValueResult => {
  if (arg.argType === "output") {
    const hasValue = !!execution?.outputs && arg.name in execution.outputs;
    return {
      hasValue,
      value: hasValue ? execution?.outputs?.[arg.name] : undefined,
    };
  }

  const hasValue = !!execution?.inputs && arg.name in execution.inputs;
  return {
    hasValue,
    value: hasValue ? execution?.inputs?.[arg.name] : undefined,
  };
};

const StepAttributesSection: React.FC<StepAttributesSectionProps> = ({
  step,
  satisfiedArgs,
  showStatus = false,
  execution,
  attributeProvenance = new Map(),
}) => {
  const unifiedArgs: UnifiedArg[] = getSortedAttributes(
    step.attributes || {}
  ).map(({ name, spec }) => ({
    name,
    type: spec.type || "any",
    argType:
      spec.role === AttributeRole.Required
        ? ("required" as const)
        : spec.role === AttributeRole.Optional
          ? ("optional" as const)
          : ("output" as const),
    spec,
  }));

  if (unifiedArgs.length === 0) {
    return null;
  }

  return (
    <div
      className={`${styles.argsSection} step-args-section`}
      data-testid="step-args"
    >
      {unifiedArgs.map((arg) => {
        const { hasValue, value } = getArgValue(arg, execution);
        const isWinner = attributeProvenance.get(arg.name) === step.id;
        const isSatisfied = satisfiedArgs.has(arg.name);

        const { Icon, className } = getArgIcon(arg.argType);

        const isProvidedByUpstream =
          arg.argType === "optional" ? isSatisfied : undefined;
        const wasDefaulted =
          arg.argType === "optional" ? hasValue && !isSatisfied : undefined;

        const statusBadge = showStatus
          ? renderStatusBadge(arg.argType, {
              isSatisfied,
              executionStatus: execution?.status,
              isWinner,
              isProvidedByUpstream,
              wasDefaulted,
            })
          : null;

        const argItem = (
          <div
            className={styles.argItem}
            data-arg-type={arg.argType}
            data-arg-name={arg.name}
          >
            <span className={styles.argName}>
              <Icon className={className} />
              {arg.name}
            </span>
            <div className={styles.argTypeContainer}>
              <span className={styles.argType}>{arg.type}</span>
              {statusBadge}
            </div>
          </div>
        );

        const key = `${arg.argType}-${arg.name}`;

        const tooltipContent = hasValue
          ? {
              title: getTooltipTitle(arg.argType, wasDefaulted),
              icon: <Icon className={`${className} ${styles.tooltipIcon}`} />,
              content: formatValue(value),
              monospace: true,
            }
          : null;

        return tooltipContent ? (
          <Tooltip key={key} trigger={argItem}>
            <TooltipSection
              title={tooltipContent.title}
              icon={tooltipContent.icon}
              monospace={tooltipContent.monospace}
            >
              {tooltipContent.content}
            </TooltipSection>
          </Tooltip>
        ) : (
          <div key={key}>{argItem}</div>
        );
      })}
    </div>
  );
};

export default React.memo(StepAttributesSection);
