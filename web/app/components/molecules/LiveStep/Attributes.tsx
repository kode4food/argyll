import React from "react";
import { Step, ExecutionResult, AttributeRole } from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import { useT } from "@/app/i18n";
import { getArgIcon } from "@/utils/iconRegistry";
import { parseInputValue } from "@/utils/inputParseUtils";
import {
  getAttributeModifiers,
  getSortedAttributes,
  ROLE_ARG_TYPE,
} from "@/utils/stepUtils";
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

interface AttributesProps {
  step: Step;
  satisfiedArgs: Set<string>;
  availableArgs?: Set<string>;
  execution?: ExecutionResult;
  // attribute name -> step ID that produced it
  attributeProvenance?: Map<string, string>;
}

interface AttributeItemProps {
  arg: UnifiedArg;
  stepId: string;
  execution?: ExecutionResult;
  attributeProvenance: Map<string, string>;
  satisfiedArgs: Set<string>;
  availableArgs: Set<string>;
}

const defaultMatchesExecutionInput = (
  rawDefault: unknown,
  executionValue: unknown
): boolean => {
  if (typeof rawDefault !== "string") return false;

  const defaultValue = parseInputValue(rawDefault);
  if (Object.is(defaultValue, executionValue)) return true;

  try {
    return JSON.stringify(defaultValue) === JSON.stringify(executionValue);
  } catch {
    return false;
  }
};

const AttributeItem: React.FC<AttributeItemProps> = ({
  arg,
  stepId,
  execution,
  attributeProvenance,
  satisfiedArgs,
  availableArgs,
}) => {
  const t = useT();
  const renderStatusBadge = useAttributeStatusBadge();

  const { hasValue, value } = getAttributeValue(arg, execution);
  const isWinner = attributeProvenance.get(arg.name) === stepId;
  const isConst = arg.argType === "const";
  const isUnsatisfied = execution?.unsatisfied?.includes(arg.name) ?? false;
  const hasExecutionDecision = !!execution && execution.status !== "pending";
  const executionInputName = getExecutionInputName(arg);
  const executionInputValue = execution?.inputs?.[executionInputName];
  const optionalUsedDefault =
    hasExecutionDecision &&
    arg.argType === "optional" &&
    !!execution?.inputs &&
    executionInputName in execution.inputs &&
    defaultMatchesExecutionInput(
      arg.spec.optional?.default,
      executionInputValue
    );
  const isSatisfied =
    hasExecutionDecision &&
    (isConst || optionalUsedDefault ? hasValue : satisfiedArgs.has(arg.name));
  const isAvailable =
    !hasExecutionDecision && !isSatisfied && availableArgs.has(arg.name);

  const { Icon, className } = getArgIcon(arg.argType);

  const isProvidedByUpstream =
    arg.argType === "optional"
      ? hasExecutionDecision &&
        !!execution?.inputs &&
        executionInputName in execution.inputs &&
        !optionalUsedDefault
      : undefined;
  const wasDefaulted =
    arg.argType === "optional"
      ? optionalUsedDefault
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
}) => {
  const unifiedArgs: UnifiedArg[] = getSortedAttributes(
    step.attributes || {}
  ).map(({ name, spec }) => ({
    name,
    type: spec.type || "any",
    argType: ROLE_ARG_TYPE[spec.role],
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
        />
      ))}
    </div>
  );
};

export default React.memo(Attributes);
