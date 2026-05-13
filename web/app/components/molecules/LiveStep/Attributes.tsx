import React from "react";
import {
  Step,
  ExecutionResult,
  AttributeRole,
  AttributeValue,
} from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import { getArgIcon } from "@/utils/iconRegistry";
import { getAttributeModifiers, getSortedAttributes } from "@/utils/stepUtils";
import {
  formatAttributeValue,
  getExecutionInputName,
  getAttributeTooltipTitle,
  getAttributeValue,
  UnifiedArg,
} from "./attributeUtils";
import { useAttributeStatusBadge } from "./useAttributeDisplay";
import ArgModifiers, { argTypeTitleKey } from "../StepShared/ArgModifiers";
import styles from "../StepShared/StepAttributesSection.module.css";
import { useT } from "@/app/i18n";

interface AttributesProps {
  step: Step;
  satisfiedArgs: Set<string>;
  availableArgs?: Set<string>;
  execution?: ExecutionResult;
  // attribute name -> step ID that produced it
  attributeProvenance?: Map<string, string>;
  attributeValues?: Record<string, AttributeValue>;
}

interface AttributeItemProps {
  arg: UnifiedArg;
  stepId: string;
  execution?: ExecutionResult;
  attributeProvenance: Map<string, string>;
  satisfiedArgs: Set<string>;
  availableArgs: Set<string>;
  attributeValues?: Record<string, AttributeValue>;
}

const hasExecutionInput = (
  execution: ExecutionResult | undefined,
  name: string
): boolean =>
  !!execution?.inputs &&
  Object.prototype.hasOwnProperty.call(execution.inputs, name);

const defaultMatchesExecutionInput = (
  rawDefault: unknown,
  executionValue: unknown
): boolean => {
  if (rawDefault === undefined) return false;

  let parsedDefault: unknown = rawDefault;
  if (typeof rawDefault === "string") {
    try {
      parsedDefault = JSON.parse(rawDefault);
    } catch {
      parsedDefault = rawDefault;
    }
  }

  if (Object.is(parsedDefault, executionValue)) return true;

  if (
    parsedDefault !== null &&
    executionValue !== null &&
    typeof parsedDefault === "object" &&
    typeof executionValue === "object"
  ) {
    try {
      return JSON.stringify(parsedDefault) === JSON.stringify(executionValue);
    } catch {
      return false;
    }
  }

  return false;
};

const AttributeItem: React.FC<AttributeItemProps> = ({
  arg,
  stepId,
  execution,
  attributeProvenance,
  satisfiedArgs,
  availableArgs,
  attributeValues,
}) => {
  const t = useT();
  const renderStatusBadge = useAttributeStatusBadge();

  const { hasValue, value } = getAttributeValue(
    arg,
    execution,
    attributeValues
  );
  const isWinner = attributeProvenance.get(arg.name) === stepId;
  const isConst = arg.argType === "const";
  const isUnsatisfied = execution?.unsatisfied?.includes(arg.name) ?? false;
  const isSatisfied = isConst ? hasValue : satisfiedArgs.has(arg.name);
  const isAvailable = !isSatisfied && availableArgs.has(arg.name);
  const executionInputName = getExecutionInputName(arg);
  const executionInputValue = execution?.inputs?.[executionInputName];
  const optionalUsedDefault =
    arg.argType === "optional" &&
    hasExecutionInput(execution, executionInputName) &&
    defaultMatchesExecutionInput(
      arg.spec.optional?.default,
      executionInputValue
    );

  const { Icon, className } = getArgIcon(arg.argType);

  const isProvidedByUpstream =
    arg.argType === "optional"
      ? hasExecutionInput(execution, executionInputName)
        ? !optionalUsedDefault
        : attributeProvenance.has(arg.name)
      : undefined;
  const wasDefaulted =
    arg.argType === "optional"
      ? optionalUsedDefault || (hasValue && !attributeProvenance.has(arg.name))
      : isConst
        ? hasValue
        : undefined;

  const statusBadge = renderStatusBadge(arg.argType, {
    isSatisfied,
    isAvailable,
    executionStatus: execution?.status,
    isUnsatisfied,
    isWinner,
    isProvidedByUpstream,
    wasDefaulted,
  });

  const argItem = (
    <div
      className={styles.argItem}
      data-arg-type={arg.argType}
      data-arg-name={arg.name}
    >
      <span className={styles.argName}>
        <span title={t(argTypeTitleKey[arg.argType])}>
          <Icon className={className} />
        </span>
        {arg.name}
        <ArgModifiers modifiers={arg.modifiers} t={t} />
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
        title: t(getAttributeTooltipTitle(arg.argType, wasDefaulted)),
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
};

const Attributes: React.FC<AttributesProps> = ({
  step,
  satisfiedArgs,
  availableArgs = new Set(),
  execution,
  attributeProvenance = new Map(),
  attributeValues,
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
          : spec.role === AttributeRole.Const
            ? ("const" as const)
            : ("output" as const),
    spec,
    modifiers: getAttributeModifiers(spec),
  }));

  if (unifiedArgs.length === 0) {
    return null;
  }

  return (
    <div
      className={`${styles.argsSection} step-args-section`}
      data-testid="step-args"
    >
      {unifiedArgs.map((arg) => (
        <AttributeItem
          key={`${arg.argType}-${arg.name}`}
          arg={arg}
          stepId={step.id}
          execution={execution}
          attributeProvenance={attributeProvenance}
          satisfiedArgs={satisfiedArgs}
          availableArgs={availableArgs}
          attributeValues={attributeValues}
        />
      ))}
    </div>
  );
};

export default React.memo(Attributes);
