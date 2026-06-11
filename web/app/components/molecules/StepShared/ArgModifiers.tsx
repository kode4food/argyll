import React from "react";
import { type ArgType } from "@/utils/iconRegistry";
import { type AttributeModifier, getModifierTitleKey } from "@/utils/stepUtils";
import { formatScriptForTooltip } from "@/utils/stepFooterUtils";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import tooltipStyles from "@/app/components/atoms/TooltipSection/TooltipSection.module.css";
import styles from "./StepAttributesSection.module.css";

interface ArgModifiersProps {
  modifiers: AttributeModifier[];
  t: (key: string, vars?: Record<string, string | number>) => string;
}

export const argTypeTitleKey: Record<ArgType, string> = {
  required: "attribute.roleRequired",
  optional: "attribute.roleOptional",
  const: "attribute.roleConst",
  meta: "attribute.roleMeta",
  output: "attribute.roleOutput",
};

const ArgModifiers: React.FC<ArgModifiersProps> = ({ modifiers, t }) => (
  <>
    {modifiers.map((mod, i) => {
      if (mod.kind === "match") {
        const { preview, lineCount } = formatScriptForTooltip(
          mod.script.script
        );
        return (
          <Tooltip
            key={i}
            trigger={
              <span>
                <mod.Icon className={styles.argModifierIcon} />
              </span>
            }
          >
            <TooltipSection
              title={t("attribute.modifierMatchTitle", {
                language: mod.script.language,
              })}
            >
              <div className={tooltipStyles.valueCode}>
                {preview}
                {lineCount > 5 && (
                  <div className={tooltipStyles.codeMore}>
                    {t("stepPredicate.moreLines", { count: lineCount - 5 })}
                  </div>
                )}
              </div>
            </TooltipSection>
          </Tooltip>
        );
      }

      if (mod.kind === "collect") {
        return (
          <span
            key={i}
            className={styles.argModifierCollect}
            title={t(getModifierTitleKey(mod))}
            style={{
              maskImage: `url(/icons/collect-${mod.collect}.svg)`,
              WebkitMaskImage: `url(/icons/collect-${mod.collect}.svg)`,
            }}
          />
        );
      }

      return (
        <span key={i} title={t(getModifierTitleKey(mod))}>
          <mod.Icon className={styles.argModifierIcon} />
        </span>
      );
    })}
  </>
);

export default ArgModifiers;
