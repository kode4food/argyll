import React from "react";
import { type AttributeModifier, getModifierTitleKey } from "@/utils/stepUtils";
import styles from "./StepAttributesSection.module.css";

interface ArgModifiersProps {
  modifiers: AttributeModifier[];
  t: (key: string) => string;
}

export const argTypeTitleKey: Record<string, string> = {
  required: "attribute.roleRequired",
  optional: "attribute.roleOptional",
  const: "attribute.roleConst",
  output: "attribute.roleOutput",
};

const ArgModifiers: React.FC<ArgModifiersProps> = ({ modifiers, t }) => (
  <>
    {modifiers.map((mod, i) =>
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
  </>
);

export default ArgModifiers;
