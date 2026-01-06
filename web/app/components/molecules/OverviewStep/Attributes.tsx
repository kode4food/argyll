import React from "react";
import { Step, AttributeRole } from "@/app/api";
import { getArgIcon } from "@/utils/argIcons";
import { getSortedAttributes } from "@/utils/stepUtils";
import styles from "../StepShared/StepAttributesSection.module.css";

interface AttributesProps {
  step: Step;
}

const Attributes: React.FC<AttributesProps> = ({ step }) => {
  const unifiedArgs = getSortedAttributes(step.attributes || {}).map(
    ({ name, spec }) => ({
      name,
      type: spec.type || "any",
      argType:
        spec.role === AttributeRole.Required
          ? ("required" as const)
          : spec.role === AttributeRole.Optional
            ? ("optional" as const)
            : ("output" as const),
    })
  );

  if (unifiedArgs.length === 0) {
    return null;
  }

  return (
    <div
      className={`${styles.argsSection} step-args-section`}
      data-testid="step-args"
    >
      {unifiedArgs.map((arg) => {
        const { Icon, className } = getArgIcon(arg.argType);
        const key = `${arg.argType}-${arg.name}`;

        return (
          <div
            key={key}
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
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default React.memo(Attributes);
