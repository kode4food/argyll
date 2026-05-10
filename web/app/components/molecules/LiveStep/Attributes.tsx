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
import {
  getAttributeModifiers,
  getModifierTitleKey,
  getSortedAttributes,
} from "@/utils/stepUtils";
import {
  formatAttributeValue,
  getAttributeTooltipTitle,
  getAttributeValue,
  UnifiedArg,
} from "./attributeUtils";
import { useAttributeStatusBadge } from "./useAttributeDisplay";
import styles from "../StepShared/StepAttributesSection.module.css";
import { useT } from "@/app/i18n";

interface AttributesProps {
  step: Step;
  satisfiedArgs: Set<string>;
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
  attributeValues?: Record<string, AttributeValue>;
}

const argTypeTitleKey: Record<string, string> = {
  required: "attribute.roleRequired",
  optional: "attribute.roleOptional",
  const: "attribute.roleConst",
  output: "attribute.roleOutput",
};

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
  const isSatisfied = isConst ? hasValue : satisfiedArgs.has(arg.name);
  const executionInputValue = execution?.inputs?.[arg.name];
  const optionalUsedDefault =
    arg.argType === "optional" &&
    hasExecutionInput(execution, arg.name) &&
    defaultMatchesExecutionInput(
      arg.spec.optional?.default,
      executionInputValue
    );

  const { Icon, className } = getArgIcon(arg.argType);

  const isProvidedByUpstream =
    arg.argType === "optional"
      ? hasExecutionInput(execution, arg.name)
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
    executionStatus: execution?.status,
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
        {arg.modifiers.map((mod, i) =>
          mod.kind === "icon" ? (
            <span key={i} title={t(getModifierTitleKey(mod))}>
              <mod.Icon className={styles.argModifierIcon} />
            </span>
          ) : (
            <span
              key={i}
              className={styles.argModifierCollect}
              title={t(getModifierTitleKey(mod))}
              style={{
                maskImage: `url(/icons/collect-${mod.collect}.svg)`,
                WebkitMaskImage: `url(/icons/collect-${mod.collect}.svg)`,
              }}
            />
          )
        )}
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
          attributeValues={attributeValues}
        />
      ))}
    </div>
  );
};

export default React.memo(Attributes);
