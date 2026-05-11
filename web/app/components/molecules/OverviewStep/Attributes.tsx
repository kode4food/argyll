import React from "react";
import { Step, AttributeRole } from "@/app/api";
import { useT } from "@/app/i18n";
import { getArgIcon } from "@/utils/iconRegistry";
import { getAttributeModifiers, getSortedAttributes } from "@/utils/stepUtils";
import ArgModifiers, { argTypeTitleKey } from "../StepShared/ArgModifiers";
import styles from "../StepShared/StepAttributesSection.module.css";

interface AttributesProps {
  step: Step;
  focusedAttributeName?: string | null;
}

const Attributes: React.FC<AttributesProps> = ({
  step,
  focusedAttributeName = null,
}) => {
  const t = useT();
  const unifiedArgs = getSortedAttributes(step.attributes || {}).map(
    ({ name, spec }) => ({
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
      modifiers: getAttributeModifiers(spec),
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
        const isFocused = focusedAttributeName === arg.name;
        const focusDirectionClass = isFocused
          ? arg.argType === "output"
            ? styles.argItemFocusedOutput
            : styles.argItemFocusedInput
          : "";

        return (
          <div
            key={key}
            className={`${styles.argItem} ${
              isFocused ? styles.argItemFocused : ""
            } ${focusDirectionClass}`}
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
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default React.memo(Attributes);
