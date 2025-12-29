import React from "react";
import {
  Step,
  ExecutionResult,
  AttributeRole,
  AttributeValue,
} from "../../api";
import Tooltip from "../atoms/Tooltip";
import TooltipSection from "../atoms/TooltipSection";
import { getArgIcon } from "@/utils/argIcons";
import { getSortedAttributes } from "@/utils/stepUtils";
import {
  formatAttributeValue,
  getAttributeTooltipTitle,
  getAttributeValue,
  UnifiedArg,
} from "./StepAttributesSection/stepAttributesSectionUtils";
import { useAttributeStatusBadge } from "./StepAttributesSection/useAttributeDisplay";
import styles from "./StepAttributesSection.module.css";

interface StepAttributesSectionProps {
  step: Step;
  satisfiedArgs: Set<string>;
  showStatus?: boolean;
  execution?: ExecutionResult;
  attributeProvenance?: Map<string, string>; // attribute name -> step ID that produced it
  attributeValues?: Record<string, AttributeValue>;
}

const StepAttributesSection: React.FC<StepAttributesSectionProps> = ({
  step,
  satisfiedArgs,
  showStatus = false,
  execution,
  attributeProvenance = new Map(),
  attributeValues,
}) => {
  const renderStatusBadge = useAttributeStatusBadge();

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
        const { hasValue, value } = getAttributeValue(
          arg,
          execution,
          attributeValues
        );
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
              title: getAttributeTooltipTitle(arg.argType, wasDefaulted),
              icon: <Icon className={`${className} ${styles.tooltipIcon}`} />,
              content: formatAttributeValue(value),
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
